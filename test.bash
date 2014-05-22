go install github.com/wangkuiyi/prism/prism
go install github.com/wangkuiyi/prism/example
go install github.com/wangkuiyi/prism/example/hello

killall prism
killall hello

# Start Prism and listen on :12340
$GOPATH/bin/prism -addr=:12340 -namenode=:50070&


# Deploy and launch hello using Prism
sleep 1
$GOPATH/bin/example -prism=:12340 -namenode=:50070 -action=launch
sleep 1
curl http://localhost:8080/Hello

# Kill hello
$GOPATH/bin/example -prism=:12340 -namenode=:50070 -action=kill
sleep 1
curl http://localhost:8080/Hello

# Deploy and launch again
$GOPATH/bin/example -prism=:12340 -namenode=:50070 -action=launch
sleep 1
curl http://localhost:8080/Hello

# Kill again
$GOPATH/bin/example -prism=:12340 -namenode=:50070 -action=kill
sleep 1
curl http://localhost:8080/Hello
