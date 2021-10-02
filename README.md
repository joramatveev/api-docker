# RUN

Step 1:

```
docker-compose up -d
```

Step 2:

```
cd api/
go mod init api-docker/api
go mod tidy
go install api-docker/api
go build
sudo ./api
```

Step 3:

```
cd ../ws/
go mod init api-docker/ws
go mod tidy
go install api-docker/ws
go build
sudo ./ws
```

<pre>
<a href="http://127.0.0.1:15672">http://127.0.0.1:15672</a> -- RabbitMQ
<a href="http://127.0.0.1:9090">http://127.0.0.1:9090</a> -- Prometheus
<a href="http://127.0.0.1:8080">http://127.0.0.1:8080</a> -- API
<a href="#">ws://127.0.0.1:8085</a> -- WebSockets
</pre>
