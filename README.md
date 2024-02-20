# Nginx Request Exporter for Prometheus

This is a [Prometheus](https://prometheus.io/) exporter for [Nginx](http://nginx.org/) requests. 

In contrast to existing exporters nginx_request_exporter does *not* scrape the [stub status module](http://nginx.org/en/docs/http/ngx_http_stub_status_module.html) for server status but records statistics for HTTP requests.

By default nginx_request_exporter listens on port 9147 for HTTP requests.

## Installation

### Using `go get`

```
git clone  git@github.com:markuslindenberg/nginx_request_exporter.git
cd nginx_request_exporter
go mod init
go mod tidy

# go get github.com/markuslindenberg/nginx_request_exporter
```

### Using Docker

```
docker pull markuslindenberg/nginx_request_exporter

公共镜像
docker run -d -p 9147:9147 -p 9514:9514/udp    markuslindenberg/nginx_request_exporter  -nginx.syslog-address=":9514"
```

## Configuration

nginx_request_exporter consumes access log records using the syslog protocol. Nginx needs to be configured to log to nginx_request_exporter's syslog port. To enable syslog logging add a `access_log` statement to your Nginx configuration:

```
access_log syslog:server=127.0.0.1:9514 prometheus;
```

## Log format

nginx_request_exporter uses a custom log format that needs to be defined in the `http` context.

The format has to only include key/value pairs:

* A key/value pair delimited by a colon denotes a metric name&value
* A key/value pair delimited by a equal sign denotes a label name&value that is added to all metrics.

Example:

```
log_format prometheus 'time:$request_time status=$status host="$host" method="$request_method" upstream="$upstream_addr"';

精简后的
log_format prometheus 'time:$request_time status=$status  method="$request_method"';

```

Multiple metrics can be recorded and all [variables](http://nginx.org/en/docs/varindex.html) available in Nginx can be used. 
Currently nginx_request_exporter has to be restarted when the log format is changed.


push

```nashorn js
create a new repository on the command line
echo "# nginx_request_exporter" >> README.md
git init
git add README.md
git commit -m "first commit"
git branch -M main
git remote add origin git@github.com:xiaomi0214/nginx_request_exporter.git
git push -u origin main

or push an existing repository from the command line
git remote add origin git@github.com:xiaomi0214/nginx_request_exporter.git
git branch -M main
git push -u origin main



or import code from another repository
You can initialize this repository with code from a Subversion, Mercurial, or TFS project.
```