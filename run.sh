#/bin/bash
#go generate . && go build -o tmp/cube && ./tmp/cube $@
go generate . && go build -o tmp/cube && ./tmp/cube -c ./tmp/config/ $@