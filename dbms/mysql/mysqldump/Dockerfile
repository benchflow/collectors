FROM benchflow/base-images:envconsul_dev

MAINTAINER Vincenzo FERME <info@vincenzoferme.it>

ENV COLLECTOR_NAME mysqldump
ENV COLLECTOR_VERSION v-dev

#TODO: pin version of mysql-client
#TODO: if we will end up using the mc for now (minio is under heavy development), we should move it to a base-image for the collectors
RUN apk --update add wget gzip mysql-client && \
    wget -q --no-check-certificate -O /app/$COLLECTOR_NAME https://github.com/benchflow/collectors/releases/download/$COLLECTOR_VERSION/$COLLECTOR_NAME && \
    chmod +x /app/$COLLECTOR_NAME && \
    wget -q --no-check-certificate -O /app/mc https://dl.minio.io/client/mc/release/linux-386/mc && \
    chmod +x /app/mc && \
    apk del --purge wget && \
    rm -rf /var/cache/apk/*

#TODO: remove, here for testing purposes
RUN /app/mc -C "/app" config host add http://195.176.181.55:9000 CYNQML6R7V12MTT32W6P SQ96V5pg02Z3kZ/0ViF9YY6GwWzZvoBmElpzEEjn

COPY ./services/300-dbms-mysql-mysqldump-collector.conf /apps/chaperone.d/300-dbms-mysql-mysqldump-collector.conf

EXPOSE 8080