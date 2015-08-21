require 'bundler/setup'
require 'rest-client'
require 'json'
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

    mixpanel = Mixpanel::Consumer.new(*%i(track engage import).map { |s| "#{@host}/#{s}" })
    error_handler = ErrorHandler.new
    @tracker = Mixpanel::Tracker.new(@token, error_handler) do |type, message|
      mixpanel.send!(type, message)
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

  #
  # var tests
  #

  def test_dummy
    assert true
  end

  #
  # mxp sink related tests
  #

  def test_mxpsink_root
    res = RestClient.get("#{@host}/")
    assert_equal 200, res.code
    assert_equal "{}", res.body
  end

end
