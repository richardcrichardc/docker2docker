# docker2docker
An experimental tool for efficiently transferring images between [Docker](https://www.docker.com/Docker) daemons. 

Example usage:

```
> docker2docker -s unix:///var/run/docker.sock -d tcp://trusty:5555 busybox
Retrieving layer info of busybox from unix:///var/run/docker.sock...
Transfering layers from unix:///var/run/docker.sock to tcp://trusty:5555...
1/4 511136ea3c... Already exists
2/4 df7546f9f0... 0.0MB / ~ 0.0MB
3/4 ea13149945... 2.5MB / ~ 2.3MB
4/4 4986bf8c15... 0.0MB / ~ 0.0MB
>

```

Docker provides tools to build, ship and run Linux containers. The shipping functionality is based around transfering container images between Docker daemons and centralised Docker registries, primarily [Docker Hub](https://registry.hub.docker.com/). Docker images can get quite large, so the the Docker [`push`](https://docs.docker.com/reference/commandline/cli/#push) and [`pull`](https://docs.docker.com/reference/commandline/cli/#pull) commands avoid redundant data transfers by only transfering image [layers](https://docs.docker.com/terms/layer/) which are not already present at the destination.

Docker supports the transfer of images between daemons without the use of a registry with the [`save`](https://docs.docker.com/reference/commandline/cli/#save) and [`load`](https://docs.docker.com/reference/commandline/cli/#load) commands. However since the `save` command has no knowledge of the destination, it must export all layers, which frequently leads to some very large and redundant data transfers. 

This tool implements an efficient transfer like Docker `push` and `pull`, for transfers between Docker daemons. It does so by querying the destination daemon and only transfering layers which are not already present.

Please note:
* **This is an experimental tool and currently relies on an [extension](http://link/to/pull/request) to the Docker API which has not and may never be included in Docker.** That said, this tool could be implemented using the current API, however it would only be efficient when run on the same machine as the source daemon. 
* **Encrypted transfers have not yet been implemented, only use this tool locally or on a network you trust.** I am planning on implementing tunnelling through SSH and may also implement SSH connections to Docker.
* As this is an experimental tool, it is only available as source code. If you are a [Go](http://golang.org/) programmer, you know what you need to do to run it. 
