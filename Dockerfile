FROM alpine:3.6

COPY server/server /app/server
COPY frontend/dist /app/public

RUN /app/server
