# Prism

Prism includes an RPC server running on each computer of a cluster.
It can deploy published binary files to local filesystems, with simple
versioning taking into consideration.  It can also start and restart
processes that execute deployed server programs.  The ability to
restart processes makes Prism a platform to build fault-recoverable
applications.

Please be aware that Prism can start only server programs, i.e.,
processes listen on specified ports.

Prism depends on package
[github.com/wangkuiyi/file](http://godoc.org/github.com/wangkuiyi/file)
to handle files across different file systems including HDFS, an
in-memory filesystem, and local filesystem.

Prism consists of a client package and a server program.  The way they
work with application programs is illustrated as follow:

                         applicaton
                         server process 1
                          ^
    applicaton            |   application        applicaton
    launcher              |   server process 2   server process 3
      |                   |      ^                 ^
     \|/                   \     |                 |
    Prism                    Prism               Prism
    client   ---- RPC ---->  server              server
    package                  process             process
            \                  ^                /  ^
             \-------- RPC ----|---------------/   |
                               |                   |
    application             application          application
    server                  server               server
    binaries                binaries             binaries
      |                        ^                   ^
      |                        |                   |
      |                        |                   |
      |                      -------------------------
      \---- Publish -------> |          HDFS         |
                             -------------------------

## The Client Package

Application programs are supposed to use package
[prism](http://github.com/wangkuiyi/prism)

1. to publish a directory of binary files (typically to HDFS), and
1. to interact with
   [Prism server](http://github.com/wangkuiyi/prism/prism) to deploy
   and launch distributed programs.

[prism.Publish](http://godoc.org/github.com/wangkuiyi/prism#Publish)
assumes that the distributed application had been build, and its
binaries (including libraries and executables) are saved in a
directory (on any filesystem supported by
[github.com/wangkuiyi/file](http://godoc.org/github.com/wangkuiyi/file).
It zip this directory and copy it to specified destination, usually on
HDFS.

[prism.Deploy](http://godoc.org/github.com/wangkuiyi/prism#Deploy)
askes the Prism server on a specified computer to download and unzip a
published zip file.  The server checks if there has been an zip file
with the same name locally.  If so, it compares the MD5 checksum of
the source zip and the local zip; if they are the same, the copying
is saved; otherwise, it overwrites the local zip.

[prism.Launch](http://godoc.org/github.com/wangkuiyi/prism#Launch)
askes the Prism server on a specified computer to start a process to
execute a specified executable file.  The caller must process a
network address (computer address and a port) to identify the process,
and, as the caller can specify command line flags, these flags should
tell the program to listen on the specified address.  The caller can
also specify the number of times that Prism server should restart the
process when it fails or crashes.  Prism server won't restart a
process that finished its work without error.

[prism.Kill](http://godoc.org/github.com/wangkuiyi/prism#Launch) kills
the process identified by a network address.  Prism server wouldn't
restart a killed process.

## The Prism Server

The Prism server is an RPC server built on top of HTTP.  To start it
on a port, say 8080:

     $GOPATH/bin/prism -prism_port=8080 -namenode=localhost:50070

where `-namenode` specifies the address of HDFS namenode.  If you are
just to test Prism locally, or you do not have HDFS installed, you can
remove that flag, and Prism server won't try to connect to HDFS.

To start Prism server on a remote computer, say `192.168.1.100`, using
SSH, in the role of `appowner`, you need to copy Prism server to that
computer, and start Prism server there:

    scp $GOPATH/bin/prism appowner@192.168.1.100:/home/appowner/
    ssh appowner@192.168.1.100 "/home/appowner/prism -prism_port=8080 \
        -namenode=192.168.1.100:50070"

Once the Prism server is started, applications can interact with it
using Prism client package.

And as Prism server is first-of-all an HTTP server, you can monitor it
using a Web browser via the URL: http://localhost:8080/debug/vars, or
http://192.168.1.100:8080/debug/vars.
