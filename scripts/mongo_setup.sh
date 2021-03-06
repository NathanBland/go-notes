#!/bin/bash
sleep 10

echo SETUP.sh time now: `date +"%T" `
mongo --host mongo1:27017 <<EOF
  var cfg = {
    "_id": "rs0",
    "version": 1,
    "members": [
      {
        "_id": 0,
        "host": "notes-mongo1:27017",
        "priority": 2
      },
      {
        "_id": 1,
        "host": "notes-mongo2:27017",
        "priority": 0
      },
      {
        "_id": 2,
        "host": "notes-mongo3:27017",
        "priority": 0
      }
    ]
  };
  rs.initiate(cfg, { force: true });
  rs.reconfig(cfg, { force: true });
  db.getMongo().setReadPref('nearest');
EOF