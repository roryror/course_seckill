#!/bin/bash

echo "start test"
curl -s http://localhost:8080/api/cleanup > /dev/null

ab -n 10000 -c 100 -k -r "http://localhost:8080/api/seckill/0/0"


echo "test done"