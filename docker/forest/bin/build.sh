export GOOS=linux
export GOARCH=amd64
wd=`pwd`
cd $GOPATH/src/github.com/andistributed/forest/forest
go build -o $wd/forest_${GOOS}_${GOARCH} -ldflags="-w -s"
cd $GOPATH/src/github.com/andistributed/duck/cmd/forester
go build -o $wd/../forest-ui/forester_${GOOS}_${GOARCH} -ldflags="-w -s"
cd $GOPATH/src/github.com/andistributed/bus/forestcmdjob
go build -o $wd/../forest-cmdjob/forestcmdjob_${GOOS}_${GOARCH} -ldflags="-w -s"
