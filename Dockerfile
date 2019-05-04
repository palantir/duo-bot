FROM scratch

ARG VERSION

ADD ca-certificates.crt /etc/ssl/certs/

ADD out/build/duo-bot/${VERSION}/linux-amd64/duo-bot /

EXPOSE 8080

CMD ["/duo-bot", "-c", "/secrets/duo-bot.yml", "server", "-a", "0.0.0.0:8080"]
