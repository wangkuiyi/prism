# Prism

Prism is an RPC server running on each computer of a cluster. It can
deploy binaries from HDFS to local filesystem and run these binaries with fault recovery.
It depends on the [file package](https://github.com/wangkuiyi/file) to handle 
files across different file systems.

## Prism Workflow

Application program utilize Prism_servers via Prism_client to launch distributed programs.


     | S1 | S2 | ... Sk |
     |------------------|
     |       HDFS       |
     |                  |

Si is the ith Prism server. 

The Prism_client bridges application and Prism_servers by providing interfaces for the 
following functions.

1. Publish
   * zip some executable files and put the package in the shared file system.
2. Deploy
   * Contact a Prism_server to copy the zip file and unpack the executables to the server's 
local directory
3. Launch
   * Contact a Prism_server to run the executables. It also assumes the executables will
listen on a predefined port for future communications
4. Kill
   * Contact a Prism_server to terminate a program


## Prism Client

1. Use the rpc.DialHTTP to connect to a Prism_server
2. Help the application to prepare and pack programs by Publish
3. Notify the server to do one of the following jobs
   * Deploy (Let the server copy the prepared program to its local dir and unpack it)
   * Launch (Let the server run the unpacked program)
   * Kill (Let the server teminate it)


## Prism Server

1. Register itself as a rpc server and listen to HTTP requests
2. Deploy (copy the executable to local. Cross filesystem file handles using [file](https://github.com/wangkuiyi/file)
3. Launch (Create a new goroutine to run the executable. If the running instance failed unintentionally, Prism will retry Launch, up to a number of times. This mechanism is critical to fault recovery and distributed computation applications) 
4. Kill (Terminate the running instance)
