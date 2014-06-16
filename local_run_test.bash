if go install github.com/wangkuiyi/prism/prism \
    github.com/wangkuiyi/prism/example \
    github.com/wangkuiyi/prism/example/hello; then
    echo -e "\033[1mBuild Prism completed\033[0m"
else
    echo -e "\033[1mBuild Prism failed\033[0m"
    exit
fi

killall prism
killall hello

# Start Prism and listen on :12340
$GOPATH/bin/prism &

SUC=0

echo -e "\033[1mDeploy and launch hello using Prism\033[0m"
sleep 1
$GOPATH/bin/example -action=launch
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo "hello is not running as expected"
    SUC=$(expr $SUC + 1)
fi

echo -e "\033[1mKill hello\033[0m"
$GOPATH/bin/example -action=kill
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != '' ]; then
    echo "hello is not killed as expected"
    SUC=$(expr $SUC + 1)
fi

echo -e "\033[1mDeploy and launch again\033[0m"
$GOPATH/bin/example -action=launch
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo "hello is not running as expected"
    SUC=$(expr $SUC + 1)
fi

echo -e "\033[1mKill again\033[0m"
$GOPATH/bin/example -action=kill
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
