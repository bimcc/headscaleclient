FROM golang:1.26.5-bookworm
RUN apt-get update && apt-get install -y --no-install-recommends openjdk-17-jdk-headless unzip curl ca-certificates && rm -rf /var/lib/apt/lists/*
ENV ANDROID_HOME=/opt/android-sdk
ENV ANDROID_NDK_HOME=/opt/android-sdk/ndk/28.2.13676358
ENV JAVA_HOME=/usr/lib/jvm/java-17-openjdk-amd64
ENV PATH=/opt/gradle/gradle-8.13/bin:/opt/android-sdk/cmdline-tools/latest/bin:/opt/android-sdk/platform-tools:${PATH}
RUN curl --fail --location --retry 3 https://dl.google.com/android/repository/commandlinetools-linux-13114758_latest.zip -o /tmp/android.zip && mkdir -p /opt/android-sdk/cmdline-tools && unzip -q /tmp/android.zip -d /opt/android-sdk/cmdline-tools && mv /opt/android-sdk/cmdline-tools/cmdline-tools /opt/android-sdk/cmdline-tools/latest && rm /tmp/android.zip
RUN yes | sdkmanager --licenses >/dev/null
RUN sdkmanager 'platforms;android-35' 'build-tools;35.0.0' 'ndk;28.2.13676358' 'platform-tools'
RUN curl --fail --location --retry 3 https://services.gradle.org/distributions/gradle-8.13-bin.zip -o /tmp/gradle.zip && curl --fail --location https://services.gradle.org/distributions/gradle-8.13-bin.zip.sha256 -o /tmp/gradle.sha256 && test "$(sha256sum /tmp/gradle.zip | cut -d ' ' -f 1)" = "$(cat /tmp/gradle.sha256)" && unzip -q /tmp/gradle.zip -d /opt/gradle && rm /tmp/gradle.zip /tmp/gradle.sha256
WORKDIR /workspace/mobile
