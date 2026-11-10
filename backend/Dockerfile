# Build menggunakan arsitektur PC agar Maven tidak perlu emulasi ARM64
FROM --platform=$BUILDPLATFORM maven:3.9-eclipse-temurin-17 AS builder

WORKDIR /app

COPY pom.xml .
RUN mvn dependency:go-offline

COPY src ./src
RUN mvn clean package -DskipTests

# Runtime mengikuti arsitektur target, yaitu ARM64 untuk STB
FROM eclipse-temurin:17-jre

RUN apt-get update \
    && apt-get install -y --no-install-recommends webp \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /app/target/*.jar app.jar

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
