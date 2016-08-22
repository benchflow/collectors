FROM benchflow/base-images:dev

MAINTAINER Vincenzo FERME <info@vincenzoferme.it>

ENV COLLECTOR_NAME stats
ENV COLLECTOR_VERSION v-dev
ENV GLIBC_VERSION=2.23-r3
ENV LANG=C.UTF-8
ENV EXPOSED_PORT 8090

RUN echo "@testing http://dl-4.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
RUN echo "http://dl-4.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories

RUN apk --update add wget curl gzip libgcc libpcap libstdc++ glib libxext libxrender ncurses-terminfo-base ncurses-terminfo ncurses-libs && \
    wget -q --no-check-certificate -O /app/$COLLECTOR_NAME https://github.com/benchflow/collectors/releases/download/$COLLECTOR_VERSION/$COLLECTOR_NAME && \
    chmod +x /app/$COLLECTOR_NAME && \
    # Install glibc for complete support of nethogs libraries
    # Avoid:
    # so:libncursesw.so.6 (missing):
    # required by:
    #              nethogs-0.8.1-r1[so:libncursesw.so.6]
    for pkg in glibc-${GLIBC_VERSION} glibc-bin-${GLIBC_VERSION} glibc-i18n-${GLIBC_VERSION}; do curl -sSL https://github.com/andyshinn/alpine-pkg-glibc/releases/download/${GLIBC_VERSION}/${pkg}.apk -o /tmp/${pkg}.apk; done && \
    apk add --allow-untrusted /tmp/*.apk && \
    rm -v /tmp/*.apk && \
    ( /usr/glibc-compat/bin/localedef --force --inputfile POSIX --charmap UTF-8 C.UTF-8 || true ) && \
    echo "export LANG=C.UTF-8" > /etc/profile.d/locale.sh && \
    /usr/glibc-compat/sbin/ldconfig /lib /usr/glibc-compat/lib && \
    apk --update add nethogs@testing && \
    # Clean up
    apk del --purge wget curl glibc-i18n && \
    rm -rf /var/cache/apk/*


COPY ./services/300-environment-stats-collector.conf /apps/chaperone.d/300-environment-stats-collector.conf

## http://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.xhtml?search=unassigned&page=4
## range 80N0 not assigned to any service as per 21 August 2016
EXPOSE 8090