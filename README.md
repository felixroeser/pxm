## pxm

**a mixpanel compatible event sink** powered by go & cassandra

### installation

If you prefer to develop within a VM and not trash your system with language runtimes and databases have a look at [my dev box](https://github.com/felixroeser/a_dev_box)

````
go get github.com/felixroeser/pxm
# find the source in /vagrant/go/src/felixroeser/pxm

cqlsh
$ CREATE KEYSPACE sink_test WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
$ CREATE KEYSPACE sink_development WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
````

### mxp compatible event sink

````
go run pxm.go --config=config/development.ini mxpsink
````

### tests

just a ruby based smoke/integration test for now

````
go run pxm.go --config=config/test.ini --cmd=drop,migrate mxpsink
cd contrib
bundle install --path vendor/bundle
ruby -Ilib:test test.rb
````

#### Running the dev box on a windows host 

The bundle install command will fail because some native extensions can't be built within the shared folder.
Use this configuration to have the gems install outside the shared folder.

````
vagrant ssh
mkdir ~/.bundles
cd /vagrant/go/src/github.com/felixroeser/pxm/contrib
printf '---\nBUNDLE_PATH: "/home/vagrant/.bundles/pxm"\nBUNDLE_DISABLE_SHARED_GEMS: "1"' >> .bundle/config
bundle install
````
