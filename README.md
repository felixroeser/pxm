## pxm

**a mixpanel compatible event sink** powered by go & cassandra

### installation

````
go get gopkg.in/gcfg.v1
go get github.com/gocql/gocql

cqlsh
$ CREATE KEYSPACE sink_development WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 };
````

### dummy

````
go run pxm.go --config=config/development.ini
````