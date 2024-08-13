FROM alpine
ENTRYPOINT ["/usr/bin/discord-file-sync"]
COPY discord-file-sync /usr/bin/discord-file-sync