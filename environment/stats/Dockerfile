FROM benchflow/base-images:envconsul_dev

MAINTAINER Vincenzo FERME <info@vincenzoferme.it>

ENV COLLECTOR_NAME stats
ENV COLLECTOR_VERSION v-dev

RUN apk --update add wget gzip && \
    wget -q --no-check-certificate -O /app/$COLLECTOR_NAME https://github.com/benchflow/collectors/releases/download/$COLLECTOR_VERSION/$COLLECTOR_NAME && \
    chmod +x /app/$COLLECTOR_NAME && \
    apk del --purge wget && \
    rm -rf /var/cache/apk/*

COPY ./services/300-environment-stats-collector.conf /apps/chaperone.d/300-environment-stats-collector.conf
#TODO: remove, here for testing purposes
RUN touch /app/test.tpl
#TODO: remove, here because of a problem killing the container
RUN rm /apps/chaperone.d/200-envconsul-envcp.conf

EXPOSE 8080