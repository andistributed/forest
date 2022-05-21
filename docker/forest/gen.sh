docker build . -t "admpub/forest:latest"
docker push admpub/forest:latest
pwd=`pwd`
cd ./forest-cmdjob
docker build . -t "admpub/forest-cmdjob:latest"
docker push admpub/forest-cmdjob:latest
cd $pwd/forest-ui
docker build . -t "admpub/forest-ui:latest"
docker push admpub/forest-ui:latest