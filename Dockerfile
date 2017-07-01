FROM alpine:3.6

COPY server/server /app/server
COPY frontend/dist /app/public

CMD /app/server
