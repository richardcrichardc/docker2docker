# docker2docker
An experimental tool for efficiently transferring images between [Docker](https://www.docker.com/Docker) daemons.

Example usage:

```
> /docker2docker -d sshunix://trusty ghost
Retrieving layer info of ghost from unix:///var/run/docker.sock...
Transfering layers from unix:///var/run/docker.sock to sshunix://trusty...
1/23 511136ea3c... Already exists
2/23 35f6dd4dd1... Already exists
...
...
21/23 3b160c2e84... 137.7MB / ~ 131.7MB
22/23 e487cc401d... 5.3MB / ~ 5.0MB
23/23 41c7a08f2a... 0.0MB / ~ 0.0MB
>

```

Docker provides tools to build, ship and run Linux containers. The shipping functionality is based around transfering container images between Docker daemons and centralised Docker registries, primarily [Docker Hub](https://registry.hub.docker.com/). Docker images can get quite large, so the the Docker [`push`](https://docs.docker.com/reference/commandline/cli/#push) and [`pull`](https://docs.docker.com/reference/commandline/cli/#pull) commands avoid redundant data transfers by only transfering image [layers](https://docs.docker.com/terms/layer/) which are not already present at the destination.

Docker supports the transfer of images between daemons without the use of a registry with the [`save`](https://docs.docker.com/reference/commandline/cli/#save) and [`load`](https://docs.docker.com/reference/commandline/cli/#load) commands. However since the `save` command has no knowledge of the destination, it must export all layers, which frequently leads to some very large and redundant data transfers.

This tool implements an efficient transfer like Docker `push` and `pull`, for transfers between Docker daemons. It does so by querying the destination daemon and only transfering layers which are not already present.

Dockers Remote API is used for connecting to the source and destination daemons. As well as connecting via local Unix domain sockets and TCP, it can also tunnel to Unix domain sockets on remote machines via ssh.

##Usage##
```
Usage: ./docker2docker [-s address] [-d address] image [...]

  -d="unix:///var/run/docker.sock": Destination docker daemon
  -s="unix:///var/run/docker.sock": Source docker daemon

Efficiently copies images between two Docker daemons. Only layers that
are not already present at the destination are transfered.

Source and destination addresses can be in the following formats:
	unix:///path/to/unix/socket
	tcp://host:port
	sshunix://[user@]host:[/path/to/unix/socket]

Unix and tcp are the usual docker transports.

Sshunix tunnels to a unix domain socket from a remote host over ssh, it
requires the 'socat' command to be installed on the remote host.

TLS/SSL is not supported.
```

##Notes##
* **This is an experimental tool and currently relies on an [extension](https://github.com/docker/docker/pull/11104) to the Docker API which has not and may never be included in Docker. You will need to compile Docker with this extension to try this tool** That said, this tool could be implemented using the current API, however it would only be efficient when run on the same machine as the source daemon.
* Only tags of the top image, when specified on the command line are transfered.
* I need to investigate compression, it is likely layers are not being compressed when transfered to the
destination.
* I am thinking of implementing a hybrid mode which will pull layers from the registry if available, otherwise transfer them from the source daemon.
* As this is an experimental tool, it is only available as source code. If you are a [Go](http://golang.org/) programmer, you know what you need to do to run it.
