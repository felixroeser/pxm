package storage

import (
	"log"
	"fmt"
	"strings"
	"github.com/gocql/gocql"
	
	"github.com/felixroeser/pxm/lib"	
)

var allTables = [...]string{"beacons", "beacons_by_did", "people", "people_updates", "aliases", "counted_beacons_by_hour", "transactions"}

func Connect() (session *gocql.Session, err error) {
	c, _ := lib.GetContext()
	log.Printf("Connecting to Cassandra: Hosts %s using Keyspace %s", strings.Join(c.Cfg.Cassandra.Hosts,","), c.Cfg.Cassandra.Keyspace )
			
	cluster := gocql.NewCluster(c.Cfg.Cassandra.Hosts...)
	cluster.Keyspace = c.Cfg.Cassandra.Keyspace
	session, err = cluster.CreateSession()
	
	return 
}

func ExecWriteQuery(stmt string, args ...interface{}) (error) {
	c, _ := lib.GetContext()
	return c.Session.Query(stmt, args...).Exec()
}

func Drop() {
	for _, t := range allTables {
		stmt := fmt.Sprintf("DROP TABLE IF EXISTS %s", t)
		if err := ExecWriteQuery(stmt); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("dropped tables", allTables)
}

func Migrate() {
	statements := [...]string{
		// FIXME beacons will have hotspots on event
		`CREATE TABLE beacons (
			event varchar,
			request_id timeuuid,
			distinct_id varchar,
			session_id varchar,
			properties map<text, text>,
			ip varchar,
			PRIMARY KEY ((event), request_id, distinct_id)
		)`,
		`CREATE TABLE beacons_by_did (
			event varchar,
			request_id timeuuid,
			distinct_id varchar,
			session_id varchar,
			properties map<text, text>,
			ip varchar,
			PRIMARY KEY ((distinct_id), request_id)
		)`,
		`CREATE TABLE counted_beacons_by_hour (
			counter_value counter,
			event varchar,
			timeframe varchar,
			PRIMARY KEY (timeframe, event)
		)`,
		`CREATE TABLE aliases (
			distinct_id varchar,
			alias varchar,
			PRIMARY KEY (distinct_id)
			)`,
		`CREATE TABLE people (
			did varchar,
			state varchar,
			rev timeuuid,
			deleted boolean,
			transactions list<timeuuid>,
			PRIMARY KEY (did)
		)`,
		`CREATE TABLE people_updates (
			did varchar,
			rid timeuuid,
			op_set map<varchar, varchar>,
			op_set_once map<varchar, varchar>,
			op_add map<varchar, double>,
			op_append map<varchar, varchar>,
			op_union map<varchar, varchar>,
			op_unset list<varchar>,
			op_delete varchar,
			PRIMARY KEY ((did), rid)
		)`,
		`CREATE TABLE transactions (
			did varchar,
			rid timeuuid,
			amount double,
			currency varchar,
			extra map<varchar, varchar>,
			PRIMARY KEY (rid)
		)`,
	}

	for _, stmt := range statements {
		if err := ExecWriteQuery(stmt); err != nil {
			log.Fatal(err)
		}
	}
}

func Flush() {
	for _, t := range allTables {
		stmt := fmt.Sprintf("TRUNCATE %s", t)
		if err := ExecWriteQuery(stmt); err != nil {
			log.Fatal(err)
		}
	}
	log.Println("flushed tables", allTables)	
}