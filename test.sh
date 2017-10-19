version3=$(docker run -d -p 6379:6379 -d redis:3-alpine)
res=$(./check_redis_go)
docker rm -f $version3 
if [[ $res =~ ^\[OK] ]];then 
  echo "OK Version3"
else 
  echo "NG Version3"
  exit 1
fi

version4=$(docker run -d -p 6379:6379 -d redis:4-alpine)
res=$(./check_redis_go)
docker rm -f $version4 
if [[ $res =~ ^\[OK] ]];then 
  echo "OK Version4"
else 
  echo "NG Version4"
  exit 1
fi
