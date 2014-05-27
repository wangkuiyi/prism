go install github.com/wangkuiyi/prism/prism
go install github.com/wangkuiyi/prism/example
go install github.com/wangkuiyi/prism/example/hello

killall prism
killall hello

# Start Prism and listen on :12340
$GOPATH/bin/prism -namenode=:50070&

SUC=0

# Deploy and launch hello using Prism
sleep 1
$GOPATH/bin/example -namenode=:50070 -action=launch
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo "hello is not running as expected"
    SUC=$(expr $SUC + 1)
fi

# Kill hello
$GOPATH/bin/example -namenode=:50070 -action=kill
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != '' ]; then
    echo "hello is not killed as expected"
    SUC=$(expr $SUC + 1)
fi

# Deploy and launch again
$GOPATH/bin/example -namenode=:50070 -action=launch
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo "hello is not running as expected"
    SUC=$(expr $SUC + 1)
fi

# Kill again
$GOPATH/bin/example -namenode=:50070 -action=kill
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != '' ]; then
    echo "hello is not killed as expected"
    SUC=$(expr $SUC + 1)
fi

if [ "$SUC" == "0" ]; then
    echo '========= Congratulations! Testing passed. ========='
else
    echo "========= " $SUC tests failed!. " ========="
fi
