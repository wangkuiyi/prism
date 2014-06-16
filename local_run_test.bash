if go install github.com/wangkuiyi/prism/prism \
    github.com/wangkuiyi/prism/example \
    github.com/wangkuiyi/prism/example/hello; then
    echo -e "\033[1mRebuilt prism, example, hello.\033[0m"
else
    echo -e "\033[1mBuild failed.\033[0m"
    exit
fi

killall prism
killall hello

# Start Prism and listen on :12340
$GOPATH/bin/prism -namenode=:50070&

SUC=0

echo -e "\033[1mDeploy and launch hello using Prism\033[0m"
sleep 1
$GOPATH/bin/example -namenode=:50070 -action=launch -retry=10
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo -e "\033[1mhello is not running as expected\033[0m"
    SUC=$(expr $SUC + 1)
else
    echo -e "\033[1mSucceeded\033[0m"
fi

echo -e "\033[1mkillall hello should not turn down hello\033[0m"
killall hello
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo -e "\033[1mhello is not running as expected\033[0m"
    SUC=$(expr $SUC + 1)
else
    echo -e "\033[1mSucceeded\033[0m"
fi

echo -e "\033[1mKill issued via Prism should kill hello.\033[0m"
$GOPATH/bin/example -namenode=:50070 -action=kill
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != '' ]; then
    echo -e "\033[1mhello is not killed as expected\033[0m"
    SUC=$(expr $SUC + 1)
else
    echo -e "\033[1mSucceeded\033[0m"
fi

echo -e "\033[1mDeploy and launch again\033[0m"
$GOPATH/bin/example -namenode=:50070 -action=launch
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != 'Hello, "/Hello"' ]; then
    echo -e "\033[1mhello is not running as expected\033[0m"
    SUC=$(expr $SUC + 1)
else
    echo -e "\033[1mSucceeded\033[0m"
fi

echo -e "\033[1mKilling the Prism should bring down hello.\033[0m"
killall prism
sleep 1
R=$(curl -s http://localhost:8080/Hello)
if [ "$R" != '' ]; then
    echo -e "\033[1mhello is not killed as expected\033[0m"
    SUC=$(expr $SUC + 1)
else
    echo -e "\033[1mSucceeded\033[0m"
fi

if [ "$SUC" == "0" ]; then
    echo -e "\033[1m========= Congratulations! Testing passed. =========\033[0m"
else
    echo -e "\033[1m========= " $SUC tests failed!. " =========\033[0m"
fi
