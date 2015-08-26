require 'bundler/setup'
require 'rest-client'
require 'json'
require 'base64'
require 'open-uri'
require 'mixpanel-ruby'
require 'minitest/autorun'
require 'cassandra'
require 'parseconfig'

class ErrorHandler
  def handle(error)
    puts "got error: #{error}"
    false
  end
end

class TestMeme < Minitest::Unit::TestCase

  def setup
    # FIXME start pxm: go run pxm.go --config=config/test.ini --cmd=drop,migrate mxpsink dummy

    @config = ParseConfig.new(File.expand_path("../../config/test.ini", __FILE__))

    @host = "http://localhost:#{@config['mxpsink']['port']}"
    @token = @config['tokens']['trusted']

    @mixpanel_consumer = Mixpanel::Consumer.new(*%i(track engage import).map { |s| "#{@host}/#{s}" })
    @mixpanel_error_handler = ErrorHandler.new
    @tracker = Mixpanel::Tracker.new(@token, @mixpanel_error_handler) do |type, message|
      @mixpanel_consumer.send!(type, message)
    end

    # http://datastax.github.io/ruby-driver/api/#cluster-class_method
    uri = URI(@config['cassandra']['uri'])
    cluster = Cassandra.cluster(hosts: [uri.host], port: uri.port)
    keyspace = uri.path.gsub('/', '')
    @session = cluster.connect(keyspace)
  end

  def teardown
    # FIXME stop pxm
  end

  def flush!
    tables = ["beacons", "beacons_by_did", "people", "aliases", "counted_beacons_by_hour", "people_updates", "transactions"]
    tables.each do |table|
      @session.execute("TRUNCATE #{table}")
    end
  end

  def find_beacons(by_did: false)
    table = by_did ? 'beacons_by_did' : 'beacons'
    @session.execute("SELECT * FROM #{table}")
  end

  def assert_beacon(h, beacon)
    assert_equal 'pi', beacon['event']
    assert_equal 'a', beacon['properties']['product']
    assert_equal 'product_page', beacon['properties']['page']
    assert_equal h[:time].to_i, beacon['request_id'].to_time.to_i if h[:time]
  end

  #
  # var tests
  #

  def test_dummy
    assert true
  end

  #
  # mxp sink related tests
  #

  def inactive_test_mxpsink_root
    res = RestClient.get("#{@host}/")
    assert_equal 200, res.code
    assert_equal "{}", res.body
  end

  def inactive_test_mxpsink_post_beacon_with_time
    flush!
    time = Time.now.to_i - 3
    response = @tracker.track('a_random_user', 'pi', {page: "product_page", product: 'a', time: time})
    assert_equal true, response
    sleep 0.1

    [find_beacons, find_beacons(by_did: true)].map(&:to_a).each do |beacons|
      assert_equal 1, beacons.size
      assert_beacon({}, beacons[0])
    end
  end

  def test_mxpsink_post_beacon_missing_time
    skip "tbi"
    response = @tracker.track('a_random_user', 'pi', {page: "a_product_page", product: 'a'})
    assert_equal true, response
  end

  def test_mxpsink_post_beacon_with_time_60_seconds_ago
    skip "tbi"
    response = @tracker.track('a_random_user', 'pi', {page: "a_product_page", product: 'a'})
    assert_equal true, response
  end
  
  def test_mxp_post_beacon_missing_token
    @tracker = Mixpanel::Tracker.new("", @mixpanel_error_handler) do |type, message|
      @mixpanel_consumer.send!(type, message)
    end
    
    response = @tracker.track('a_random_user', 'pi', {page: "a_product_page", product: 'a'})
    assert_equal false, response
  end
  
  def test_mxp_post_beacon_untrusted_token_for_trusted
    @tracker = Mixpanel::Tracker.new("12345", @mixpanel_error_handler) do |type, message|
      @mixpanel_consumer.send!(type, message)
    end
    
    response = @tracker.track('a_random_user', 'signed_up', {})
    assert_equal false, response
  end
  
  def test_mxp_post_beacon_admin_token_for_trusted
    @tracker = Mixpanel::Tracker.new("alsosupersecret", @mixpanel_error_handler) do |type, message|
      @mixpanel_consumer.send!(type, message)
    end
    
    assert @tracker.track('a_random_user', 'signed_up', {})
  end

  def test_mxp_post_beacon_admin_token_for_admin
    @tracker = Mixpanel::Tracker.new("alsosupersecret", @mixpanel_error_handler) do |type, message|
      @mixpanel_consumer.send!(type, message)
    end
    
    assert @tracker.track('a_random_user', 'flush', {})
  end
  
  def test_mxp_post_beacon_trusted_token_for_trusted
    assert !@tracker.track('a_random_user', 'flush', {})
  end
  
  def test_mxp_post_beacon_trusted_token_for_untrusted
    assert !@tracker.track('a_random_user', 'flush', {})
  end


  def test_mxpsink_post_beacon_missing_token
    skip "tbi"
    response = @tracker.track('a_random_user', 'pi', {page: "a_product_page", product: 'a'})
    assert_equal true, response
  end

  def test_mxpsink_post_beacon_wrong_token
    skip "tbi"
    response = @tracker.track('a_random_user', 'pi', {page: "a_product_page", product: 'a'})
    assert_equal true, response
  end

  def test_mxpsink_get_beacon
    flush!

    data = {
      event: "pi",
      properties: {
        distinct_id: "a_random_user",
        token: "bbbb",
        time: 1440169735,
        page: "product_page",
        product: "a"
      }
    }

    params = {
      data: Base64.encode64(data.to_json),
      verbose: 1
    }

    response = RestClient.get("#{@host}/track", params: params)
    assert_equal 200, response.code
    assert_equal 1, JSON.parse(response.body)['status']
    assert       JSON.parse(response.body)['error'].size == 0
    sleep 0.1

    [find_beacons, find_beacons(by_did: true)].map(&:to_a).each do |beacons|
      assert_equal 1, beacons.size
      assert_beacon({}, beacons[0])
    end
  end

  def test_mxpsink_get_multiple_beacons
    skip "tbi"
  end

end
