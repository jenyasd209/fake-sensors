# Sensor

## Before run

Before run an `.env` file should be created for docker containers. The next env vars should be defined:

- POSTGRES_DB - name of sensors database;
- POSTGRES_USER - postgres user;
- POSTGRES_PASSWORD - postgres user password;
- POSTGRES_HOST - host ip for connection to postgres db;
- POSTGRES_PORT - postgres port 5432 to expose;
- REDIS_PORT=6379 - redis port to expose;
- REDIS_HOST - host ip for connection to redis;
- SENSOR_PORT - sensors service port to expose.

Example:
```shell
POSTGRES_DB=sensor
POSTGRES_USER=postgres
POSTGRES_PASSWORD=pwd
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

REDIS_HOST=redis
REDIS_PORT=6379

SENSOR_PORT=8080
```

## Run

```shell
docker-compose --env-file .env up
```

### Rebuild

```shell
docker-compose --env-file .env up --build --force-recreate
```

## After run

Visit the http://localhost:8080/swagger/index.html to check the swagger documentation for exist routes.
