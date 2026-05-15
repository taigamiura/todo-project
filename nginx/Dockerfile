ARG NGINX_VERSION=1.27.3
ARG OTEL_CPP_VERSION=1.17.0

FROM ubuntu:24.04 AS build

ARG NGINX_VERSION
ARG OTEL_CPP_VERSION

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && \
    apt-get install --no-install-recommends --no-install-suggests -y \
    build-essential \
    autoconf \
    libtool \
    pkg-config \
    ca-certificates \
    gcc \
    g++ \
    git \
    cmake \
    libcurl4-openssl-dev \
    libpcre3-dev \
    libssl-dev \
    curl \
    zlib1g-dev \
    libprotobuf-dev \
    protobuf-compiler && \
    rm -rf /var/lib/apt/lists/*

RUN git clone --shallow-submodules --depth 1 --recurse-submodules -b v${OTEL_CPP_VERSION} \
    https://github.com/open-telemetry/opentelemetry-cpp.git /opentelemetry-cpp && \
    cd /opentelemetry-cpp && \
    mkdir build && \
    cd build && \
    cmake -DCMAKE_BUILD_TYPE=Release \
    -DWITH_OTLP_HTTP=ON \
    -DWITH_OTLP_GRPC=OFF \
    -DWITH_PROMETHEUS=OFF \
    -DWITH_ELASTICSEARCH=OFF \
    -DBUILD_TESTING=OFF \
    -DWITH_EXAMPLES=OFF \
    -DWITH_FUNC_TESTS=OFF \
    -DOPENTELEMETRY_INSTALL=ON \
    -DCMAKE_CXX_STANDARD=17 \
    -DBUILD_SHARED_LIBS=OFF \
    -DCMAKE_POSITION_INDEPENDENT_CODE=ON \
    .. && \
    cmake --build . --target all -j$(nproc) && \
    cmake --install . --prefix /opentelemetry-cpp/install

RUN git clone --depth 1 https://github.com/open-telemetry/opentelemetry-cpp-contrib.git /opentelemetry-cpp-contrib && \
    mkdir -p /otel-nginx && \
    cp -R /opentelemetry-cpp-contrib/instrumentation/nginx/. /otel-nginx/

RUN mkdir -p /otel-nginx/build && \
    cd /otel-nginx/build && \
    cmake -DCMAKE_BUILD_TYPE=Release \
    -DNGINX_VERSION=${NGINX_VERSION} \
    -DCMAKE_PREFIX_PATH=/opentelemetry-cpp/install \
    -DCMAKE_INSTALL_PREFIX=/otel-nginx/install \
    .. && \
    make -j$(nproc) && \
    make install

RUN curl -O https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz && \
    tar xf nginx-${NGINX_VERSION}.tar.gz && \
    cd nginx-${NGINX_VERSION} && \
    ./configure --with-compat --with-http_ssl_module --with-http_v2_module --with-http_stub_status_module --prefix=/nginx && \
    make -j$(nproc) && \
    make install

FROM build AS run

RUN groupadd --system nginx && \
    useradd --system --gid nginx --no-create-home --shell /usr/sbin/nologin nginx && \
    mkdir -p /usr/lib/nginx/modules /var/log/nginx /var/cache/nginx /etc/nginx/certs

COPY --from=build /otel-nginx/install/otel_ngx_module.so /usr/lib/nginx/modules/otel_ngx_module.so
COPY --from=build /nginx/conf/mime.types /etc/nginx/mime.types

CMD ["/nginx/sbin/nginx", "-g", "daemon off;", "-c", "/etc/nginx/nginx.conf"]
