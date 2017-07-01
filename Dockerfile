FROM debian:8

COPY server/server /app/server
COPY frontend/dist /app/public

CMD /app/server
