FROM debian:8

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY server/server /app/server
COPY frontend/dist /app/public

CMD /app/server
