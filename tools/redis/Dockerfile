FROM redis:8.2-m01

RUN apt-get update && apt-get install -y gettext && apt-get clean

COPY redis.conf.template /etc/redis/redis.conf.template
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
