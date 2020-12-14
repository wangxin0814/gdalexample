FROM osgeo/gdal:alpine-ultrasmall-3.0.2

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
RUN apk --no-cache add ca-certificates tzdata libc6-compat

RUN mkdir -p /home/tif

RUN ln -s /usr/lib/libgdal.so.3.0.2  /usr/lib/libgdal.so.26
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig/
ENV LD_LIBRARY_PATH=/usr/lib/

COPY build/gdalexample /usr/bin/gdalexample
ENTRYPOINT ["/usr/bin/gdalexample"]