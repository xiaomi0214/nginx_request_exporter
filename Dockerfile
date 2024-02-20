FROM golang:alpine
#FROM golang

RUN mkdir -p /go/src/nginx_request_exporter
WORKDIR /go/src/nginx_request_exporter
ENV GOPROXY=https://goproxy.cn,direct

COPY . /go/src/nginx_request_exporter
#RUN apk add --no-cache git \
RUN go mod download \
    && go install .
#RUN apk add --no-cache --virtual .git git ; go-wrapper download ; apk del .git
#RUN go-wrapper install

EXPOSE 9147 9514/udp
USER nobody
ENTRYPOINT ["nginx_request_exporter"]
