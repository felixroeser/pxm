## pxm

**a mixpanel compatible event sink** powered by go & cassandra

### installation

````
go get gopkg.in/gcfg.v1
go get github.com/gocql/gocql

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
