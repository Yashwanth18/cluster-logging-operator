package kafka

const (
	initKafkaScript = `
      #!/bin/bash
      set -e
      cp /etc/kafka-configmap/log4j.properties /etc/kafka/

      KAFKA_BROKER_ID=${HOSTNAME##*-}
      SEDS=("s/#init#broker.id=#init#/broker.id=$KAFKA_BROKER_ID/")
      LABELS="kafka-broker-id=$KAFKA_BROKER_ID"
      ANNOTATIONS=""

      hash kubectl 2>/dev/null || {
        SEDS+=("s/#init#broker.rack=#init#/#init#broker.rack=# kubectl not found in path/")
      } && {
        ZONE=$(kubectl get node "$NODE_NAME" -o=go-template='{{index .metadata.labels "failure-domain.beta.kubernetes.io/zone"}}')
        if [ "x$ZONE" == "x<no value>" ]; then
          SEDS+=("s/#init#broker.rack=#init#/#init#broker.rack=# zone label not found for node $NODE_NAME/")
        else
          SEDS+=("s/#init#broker.rack=#init#/broker.rack=$ZONE/")
          LABELS="$LABELS kafka-broker-rack=$ZONE"
        fi

        [ -z "$ADVERTISE_ADDR" ] && echo "ADVERTISE_ADDR is empty, will advertise detected DNS name"
        SEDS+=("s|#init#advertised.listeners=PLAINTEXT://#init#|advertised.listeners=PLAINTEXT://${ADVERTISE_ADDR}:9092,SSL://${ADVERTISE_ADDR}:9093|")

        if [ ! -z "$LABELS" ]; then
          kubectl -n $POD_NAMESPACE label pod $POD_NAME $LABELS || echo "Failed to label $POD_NAMESPACE.$POD_NAME - RBAC issue?"
        fi
        if [ ! -z "$ANNOTATIONS" ]; then
          kubectl -n $POD_NAMESPACE annotate pod $POD_NAME $ANNOTATIONS || echo "Failed to annotate $POD_NAMESPACE.$POD_NAME - RBAC issue?"
        fi
      }
      printf '%s\n' "${SEDS[@]}" | sed -f - /etc/kafka-configmap/server.properties > /etc/kafka/server.properties.tmp
      [ $? -eq 0 ] && mv /etc/kafka/server.properties.tmp /etc/kafka/server.properties

      rm -rf /var/lib/kafka/data/*
    `

	clientProperties = `
      security.protocol=SSL
      ssl.truststore.location=/etc/kafka-certs/ca-bundle.jks
      ssl.truststore.type=JKS
      ssl.truststore.password=ca-bundle
    `

	serverProperties = `
      ############################# Log Basics #############################

      # A comma separated list of directories under which to store log files
      # Overrides log.dir
      log.dirs=/var/lib/kafka/data/topics

      # The default number of log partitions per topic. More partitions allow greater
      # parallelism for consumption, but this will also result in more files across
      # the brokers.
      num.partitions=12

      default.replication.factor=1

      min.insync.replicas=1

      auto.create.topics.enable=false

      # Max Messages in Bytes set to 10M > fluentd buffer chunk_limit_size config
      message.max.bytes=10000000

      # The number of threads per data directory to be used for log recovery at startup and flushing at shutdown.
      # This value is recommended to be increased for installations with data dirs located in RAID array.
      #num.recovery.threads.per.data.dir=1

      ############################# Server Basics #############################

      # The id of the broker. This must be set to a unique integer for each broker.
      #init#broker.id=#init#

      #init#broker.rack=#init#

      ############################# Socket Server Settings #############################

      # The address the socket server listens on. It will get the value returned from
      # java.net.InetAddress.getCanonicalHostName() if not configured.
      #   FORMAT:
      #     listeners = listener_name://host_name:port
      #   EXAMPLE:
      #     listeners = PLAINTEXT://your.host.name:9092
      #listeners=PLAINTEXT://:9092
      listeners=PLAINTEXT://:9092,SSL://:9093
      ssl.keystore.type=JKS
      ssl.keystore.location=/etc/kafka-certs/server.jks
      ssl.keystore.password=server

      # Hostname and port the broker will advertise to producers and consumers. If not set,
      # it uses the value for "listeners" if configured.  Otherwise, it will use the value
      # returned from java.net.InetAddress.getCanonicalHostName().
      #advertised.listeners=PLAINTEXT://your.host.name:9092
      #init#advertised.listeners=PLAINTEXT://#init#

      # Maps listener names to security protocols, the default is for them to be the same. See the config documentation for more details
      #listener.security.protocol.map=PLAINTEXT:PLAINTEXT,SSL:SSL,SASL_PLAINTEXT:SASL_PLAINTEXT,SASL_SSL:SASL_SSL
      listener.security.protocol.map=PLAINTEXT:PLAINTEXT,SSL:SSL,SASL_PLAINTEXT:SASL_PLAINTEXT,SASL_SSL:SASL_SSL,OUTSIDE:PLAINTEXT
      inter.broker.listener.name=PLAINTEXT

      # The number of threads that the server uses for receiving requests from the network and sending responses to the network
      #num.network.threads=3

      # The number of threads that the server uses for processing requests, which may include disk I/O
      #num.io.threads=8

      # The send buffer (SO_SNDBUF) used by the socket server
      #socket.send.buffer.bytes=102400

      # The receive buffer (SO_RCVBUF) used by the socket server
      #socket.receive.buffer.bytes=102400

      # The maximum size of a request that the socket server will accept (protection against OOM)
      #socket.request.max.bytes=104857600

      ############################# Internal Topic Settings  #############################
      # The replication factor for the group metadata internal topics "__consumer_offsets" and "__transaction_state"
      # For anything other than development testing, a value greater than 1 is recommended for to ensure availability such as 3.
      offsets.topic.replication.factor=1
      transaction.state.log.replication.factor=1
      transaction.state.log.min.isr=1

      ############################# Log Flush Policy #############################

      # Messages are immediately written to the filesystem but by default we only fsync() to sync
      # the OS cache lazily. The following configurations control the flush of data to disk.
      # There are a few important trade-offs here:
      #    1. Durability: Unflushed data may be lost if you are not using replication.
      #    2. Latency: Very large flush intervals may lead to latency spikes when the flush does occur as there will be a lot of data to flush.
      #    3. Throughput: The flush is generally the most expensive operation, and a small flush interval may lead to excessive seeks.
      # The settings below allow one to configure the flush policy to flush data after a period of time or
      # every N messages (or both). This can be done globally and overridden on a per-topic basis.

      # The number of messages to accept before forcing a flush of data to disk
      #log.flush.interval.messages=10000

      # The maximum amount of time a message can sit in a log before we force a flush
      #log.flush.interval.ms=1000

      ############################# Log Retention Policy #############################

      # The following configurations control the disposal of log segments. The policy can
      # be set to delete segments after a period of time, or after a given size has accumulated.
      # A segment will be deleted whenever *either* of these criteria are met. Deletion always happens
      # from the end of the log.

      # https://cwiki.apache.org/confluence/display/KAFKA/KIP-186%3A+Increase+offsets+retention+default+to+7+days
      offsets.retention.minutes=10080

      # The minimum age of a log file to be eligible for deletion due to age
      log.retention.hours=-1

      # A size-based retention policy for logs. Segments are pruned from the log unless the remaining
      # segments drop below log.retention.bytes. Functions independently of log.retention.hours.
      #log.retention.bytes=1073741824

      # The maximum size of a log segment file. When this size is reached a new log segment will be created.
      #log.segment.bytes=1073741824

      # The interval at which log segments are checked to see if they can be deleted according
      # to the retention policies
      #log.retention.check.interval.ms=300000

      ############################# Zookeeper #############################

      # Zookeeper connection string (see zookeeper docs for details).
      # This is a comma separated host:port pairs, each corresponding to a zk
      # server. e.g. "127.0.0.1:3000,127.0.0.1:3001,127.0.0.1:3002".
      # You can also append an optional chroot string to the urls to specify the
      # root directory for all kafka znodes.
      zookeeper.connect=zookeeper.openshift-logging.svc.cluster.local:2181

      # Timeout in ms for connecting to zookeeper
      #zookeeper.connection.timeout.ms=6000


      ############################# Group Coordinator Settings #############################

      # The following configuration specifies the time, in milliseconds, that the GroupCoordinator will delay the initial consumer rebalance.
      # The rebalance will be further delayed by the value of group.initial.rebalance.delay.ms as new members join the group, up to a maximum of max.poll.interval.ms.
      # The default value for this is 3 seconds.
      # We override this to 0 here as it makes for a better out-of-the-box experience for development and testing.
      # However, in production environments the default value of 3 seconds is more suitable as this will help to avoid unnecessary, and potentially expensive, rebalances during application startup.
      #group.initial.rebalance.delay.ms=0
    `

	log4jProperties = `
      # Unspecified loggers and loggers with additivity=true output to server.log and stdout
      # Note that INFO only applies to unspecified loggers, the log level of the child logger is used otherwise
      log4j.rootLogger=INFO, stdout

      log4j.appender.stdout=org.apache.log4j.ConsoleAppender
      log4j.appender.stdout.layout=org.apache.log4j.PatternLayout
      log4j.appender.stdout.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.kafkaAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.kafkaAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.kafkaAppender.File=${kafka.logs.dir}/server.log
      log4j.appender.kafkaAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.kafkaAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.stateChangeAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.stateChangeAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.stateChangeAppender.File=${kafka.logs.dir}/state-change.log
      log4j.appender.stateChangeAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.stateChangeAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.requestAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.requestAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.requestAppender.File=${kafka.logs.dir}/kafka-request.log
      log4j.appender.requestAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.requestAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.cleanerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.cleanerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.cleanerAppender.File=${kafka.logs.dir}/log-cleaner.log
      log4j.appender.cleanerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.cleanerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.controllerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.controllerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.controllerAppender.File=${kafka.logs.dir}/controller.log
      log4j.appender.controllerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.controllerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.authorizerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.authorizerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.authorizerAppender.File=${kafka.logs.dir}/kafka-authorizer.log
      log4j.appender.authorizerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.authorizerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      # Change the two lines below to adjust ZK client logging
      log4j.logger.org.I0Itec.zkclient.ZkClient=INFO
      log4j.logger.org.apache.zookeeper=INFO

      # Change the two lines below to adjust the general broker logging level (output to server.log and stdout)
      log4j.logger.kafka=INFO
      log4j.logger.org.apache.kafka=INFO

      # Change to DEBUG or TRACE to enable request logging
      log4j.logger.kafka.request.logger=WARN, requestAppender
      log4j.additivity.kafka.request.logger=false

      # Uncomment the lines below and change log4j.logger.kafka.network.RequestChannel$ to TRACE for additional output
      # related to the handling of requests
      #log4j.logger.kafka.network.Processor=TRACE, requestAppender
      #log4j.logger.kafka.server.KafkaApis=TRACE, requestAppender
      #log4j.additivity.kafka.server.KafkaApis=false
      log4j.logger.kafka.network.RequestChannel$=WARN, requestAppender
      log4j.additivity.kafka.network.RequestChannel$=false

      log4j.logger.kafka.controller=TRACE, controllerAppender
      log4j.additivity.kafka.controller=false

      log4j.logger.kafka.log.LogCleaner=INFO, cleanerAppender
      log4j.additivity.kafka.log.LogCleaner=false

      log4j.logger.state.change.logger=TRACE, stateChangeAppender
      log4j.additivity.state.change.logger=false

      # Change to DEBUG to enable audit log for the authorizer
      log4j.logger.kafka.authorizer.logger=WARN, authorizerAppender
      log4j.additivity.kafka.authorizer.logger=false
    `

	initZookeeperScript = `
      #!/bin/bash
      set -e

      [ -d /var/lib/zookeeper/data ] || mkdir /var/lib/zookeeper/data
      [ -z "$ID_OFFSET" ] && ID_OFFSET=1
      export ZOOKEEPER_SERVER_ID=$((${HOSTNAME##*-} + $ID_OFFSET))
      echo "${ZOOKEEPER_SERVER_ID:-1}" | tee /var/lib/zookeeper/data/myid
      cp -Lur /etc/kafka-configmap/* /etc/kafka/
    `

	zookeeperProperties = `
      4lw.commands.whitelist=ruok
      tickTime=2000
      dataDir=/var/lib/zookeeper/data
      dataLogDir=/var/lib/zookeeper/log
      clientPort=2181
    `
	zookeeperLog4JProperties = `
      log4j.rootLogger=INFO, stdout
      log4j.appender.stdout=org.apache.log4j.ConsoleAppender
      log4j.appender.stdout.layout=org.apache.log4j.PatternLayout
      log4j.appender.stdout.layout.ConversionPattern=[%d] %p %m (%c)%n

      # Suppress connection log messages, three lines per livenessProbe execution
      log4j.logger.org.apache.zookeeper.server.NIOServerCnxnFactory=WARN
      log4j.logger.org.apache.zookeeper.server.NIOServerCnxn=WARN
    `
	functionalPodinitKafkaScript = `
      #!/bin/bash
      set -e
      cp /etc/kafka-configmap/log4j.properties /etc/kafka/

      KAFKA_BROKER_ID=${HOSTNAME##*-}
      #SEDS=("s/#init#broker.id=#init#/broker.id=$KAFKA_BROKER_ID/")
      LABELS="kafka-broker-id=$KAFKA_BROKER_ID"
      ANNOTATIONS=""

    `

	functionalPodclientProperties = `
      security.protocol=SSL
      ssl.truststore.location=/var/run/ocp-collector/secrets/kafka/ca-bundle.jks
      ssl.truststore.type=JKS
      ssl.truststore.password=ca-bundle
    `

	functionalPodserverProperties = `
      ############################# Log Basics #############################

      # A comma separated list of directories under which to store log files
      # Overrides log.dir
      log.dirs=/var/lib/kafka/data/topics

      # The default number of log partitions per topic. More partitions allow greater
      # parallelism for consumption, but this will also result in more files across
      # the brokers.
      num.partitions=1

      default.replication.factor=1

      min.insync.replicas=1

      auto.create.topics.enable=false

      # Max Messages in Bytes set to 10M > fluentd buffer chunk_limit_size config
      message.max.bytes=10000000

      # The number of threads per data directory to be used for log recovery at startup and flushing at shutdown.
      # This value is recommended to be increased for installations with data dirs located in RAID array.
      #num.recovery.threads.per.data.dir=1

      ############################# Server Basics #############################

      # The id of the broker. This must be set to a unique integer for each broker.
      broker.id=0

      #init#broker.rack=#init#

      ############################# Socket Server Settings #############################

      # The address the socket server listens on. It will get the value returned from
      # java.net.InetAddress.getCanonicalHostName() if not configured.
      #   FORMAT:
      #     listeners = listener_name://host_name:port
      #   EXAMPLE:
      #     listeners = PLAINTEXT://your.host.name:9092

      listeners=PLAINTEXT://:9092,SSL://:9093
      ssl.keystore.type=JKS
      ssl.keystore.location=/var/run/ocp-collector/secrets/kafka/server.jks
      ssl.keystore.password=server

      # Hostname and port the broker will advertise to producers and consumers. If not set,
      # it uses the value for "listeners" if configured.  Otherwise, it will use the value
      # returned from java.net.InetAddress.getCanonicalHostName().
      advertised.listeners=PLAINTEXT://localhost:9092,SSL://localhost:9093
      #init#advertised.listeners=PLAINTEXT://#init#

      # Maps listener names to security protocols, the default is for them to be the same. See the config documentation for more details
      #listener.security.protocol.map=PLAINTEXT:PLAINTEXT,SSL:SSL,SASL_PLAINTEXT:SASL_PLAINTEXT,SASL_SSL:SASL_SSL
      listener.security.protocol.map=PLAINTEXT:PLAINTEXT,SSL:SSL,SASL_PLAINTEXT:SASL_PLAINTEXT,SASL_SSL:SASL_SSL,OUTSIDE:PLAINTEXT
      inter.broker.listener.name=PLAINTEXT
      

      # The number of threads that the server uses for receiving requests from the network and sending responses to the network
      #num.network.threads=3

      # The number of threads that the server uses for processing requests, which may include disk I/O
      #num.io.threads=8

      # The send buffer (SO_SNDBUF) used by the socket server
      #socket.send.buffer.bytes=102400

      # The receive buffer (SO_RCVBUF) used by the socket server
      #socket.receive.buffer.bytes=102400

      # The maximum size of a request that the socket server will accept (protection against OOM)
      #socket.request.max.bytes=104857600

      ############################# Internal Topic Settings  #############################
      # The replication factor for the group metadata internal topics "__consumer_offsets" and "__transaction_state"
      # For anything other than development testing, a value greater than 1 is recommended for to ensure availability such as 3.
      offsets.topic.replication.factor=1
      transaction.state.log.replication.factor=1
      transaction.state.log.min.isr=1

      ############################# Log Flush Policy #############################

      # Messages are immediately written to the filesystem but by default we only fsync() to sync
      # the OS cache lazily. The following configurations control the flush of data to disk.
      # There are a few important trade-offs here:
      #    1. Durability: Unflushed data may be lost if you are not using replication.
      #    2. Latency: Very large flush intervals may lead to latency spikes when the flush does occur as there will be a lot of data to flush.
      #    3. Throughput: The flush is generally the most expensive operation, and a small flush interval may lead to excessive seeks.
      # The settings below allow one to configure the flush policy to flush data after a period of time or
      # every N messages (or both). This can be done globally and overridden on a per-topic basis.

      # The number of messages to accept before forcing a flush of data to disk
      #log.flush.interval.messages=10000

      # The maximum amount of time a message can sit in a log before we force a flush
      #log.flush.interval.ms=1000

      ############################# Log Retention Policy #############################

      # The following configurations control the disposal of log segments. The policy can
      # be set to delete segments after a period of time, or after a given size has accumulated.
      # A segment will be deleted whenever *either* of these criteria are met. Deletion always happens
      # from the end of the log.

      # https://cwiki.apache.org/confluence/display/KAFKA/KIP-186%3A+Increase+offsets+retention+default+to+7+days
      offsets.retention.minutes=10080

      # The minimum age of a log file to be eligible for deletion due to age
      log.retention.hours=-1

      # A size-based retention policy for logs. Segments are pruned from the log unless the remaining
      # segments drop below log.retention.bytes. Functions independently of log.retention.hours.
      #log.retention.bytes=1073741824

      # The maximum size of a log segment file. When this size is reached a new log segment will be created.
      #log.segment.bytes=1073741824

      # The interval at which log segments are checked to see if they can be deleted according
      # to the retention policies
      #log.retention.check.interval.ms=300000

      ############################# Zookeeper #############################

      # Zookeeper connection string (see zookeeper docs for details).
      # This is a comma separated host:port pairs, each corresponding to a zk
      # server. e.g. "127.0.0.1:3000,127.0.0.1:3001,127.0.0.1:3002".
      # You can also append an optional chroot string to the urls to specify the
      # root directory for all kafka znodes.
      zookeeper.connect=localhost:2181

      # Timeout in ms for connecting to zookeeper
      zookeeper.connection.timeout.ms=60000


      ############################# Group Coordinator Settings #############################

      # The following configuration specifies the time, in milliseconds, that the GroupCoordinator will delay the initial consumer rebalance.
      # The rebalance will be further delayed by the value of group.initial.rebalance.delay.ms as new members join the group, up to a maximum of max.poll.interval.ms.
      # The default value for this is 3 seconds.
      # We override this to 0 here as it makes for a better out-of-the-box experience for development and testing.
      # However, in production environments the default value of 3 seconds is more suitable as this will help to avoid unnecessary, and potentially expensive, rebalances during application startup.
      #group.initial.rebalance.delay.ms=0
    `

	functionalPodlog4jProperties = `
      # Unspecified loggers and loggers with additivity=true output to server.log and stdout
      # Note that INFO only applies to unspecified loggers, the log level of the child logger is used otherwise
      log4j.rootLogger=INFO, stdout

      log4j.appender.stdout=org.apache.log4j.ConsoleAppender
      log4j.appender.stdout.layout=org.apache.log4j.PatternLayout
      log4j.appender.stdout.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.kafkaAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.kafkaAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.kafkaAppender.File=${kafka.logs.dir}/server.log
      log4j.appender.kafkaAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.kafkaAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.stateChangeAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.stateChangeAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.stateChangeAppender.File=${kafka.logs.dir}/state-change.log
      log4j.appender.stateChangeAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.stateChangeAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.requestAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.requestAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.requestAppender.File=${kafka.logs.dir}/kafka-request.log
      log4j.appender.requestAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.requestAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.cleanerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.cleanerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.cleanerAppender.File=${kafka.logs.dir}/log-cleaner.log
      log4j.appender.cleanerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.cleanerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.controllerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.controllerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.controllerAppender.File=${kafka.logs.dir}/controller.log
      log4j.appender.controllerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.controllerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      log4j.appender.authorizerAppender=org.apache.log4j.DailyRollingFileAppender
      log4j.appender.authorizerAppender.DatePattern='.'yyyy-MM-dd-HH
      log4j.appender.authorizerAppender.File=${kafka.logs.dir}/kafka-authorizer.log
      log4j.appender.authorizerAppender.layout=org.apache.log4j.PatternLayout
      log4j.appender.authorizerAppender.layout.ConversionPattern=[%d] %p %m (%c)%n

      # Change the two lines below to adjust ZK client logging
      log4j.logger.org.I0Itec.zkclient.ZkClient=INFO
      log4j.logger.org.apache.zookeeper=INFO

      # Change the two lines below to adjust the general broker logging level (output to server.log and stdout)
      log4j.logger.kafka=INFO
      log4j.logger.org.apache.kafka=INFO

      # Change to DEBUG or TRACE to enable request logging
      log4j.logger.kafka.request.logger=WARN, requestAppender
      log4j.additivity.kafka.request.logger=false

      # Uncomment the lines below and change log4j.logger.kafka.network.RequestChannel$ to TRACE for additional output
      # related to the handling of requests
      #log4j.logger.kafka.network.Processor=TRACE, requestAppender
      #log4j.logger.kafka.server.KafkaApis=TRACE, requestAppender
      #log4j.additivity.kafka.server.KafkaApis=false
      log4j.logger.kafka.network.RequestChannel$=WARN, requestAppender
      log4j.additivity.kafka.network.RequestChannel$=false

      log4j.logger.kafka.controller=TRACE, controllerAppender
      log4j.additivity.kafka.controller=false

      log4j.logger.kafka.log.LogCleaner=INFO, cleanerAppender
      log4j.additivity.kafka.log.LogCleaner=false

      log4j.logger.state.change.logger=TRACE, stateChangeAppender
      log4j.additivity.state.change.logger=false

      # Change to DEBUG to enable audit log for the authorizer
      log4j.logger.kafka.authorizer.logger=WARN, authorizerAppender
      log4j.additivity.kafka.authorizer.logger=false
    `

	functionalPodinitZookeeperScript = `
      #!/bin/bash
      set -e

      [ -d /var/lib/zookeeper/data ] || mkdir /var/lib/zookeeper/data
      [ -z "$ID_OFFSET" ] && ID_OFFSET=1
      export ZOOKEEPER_SERVER_ID=$((${HOSTNAME##*-} + $ID_OFFSET))
      echo "${ZOOKEEPER_SERVER_ID:-1}" | tee /var/lib/zookeeper/data/myid
      cp -Lur /etc/kafka-configmap/* /etc/kafka/
    `

	functionalPodzookeeperProperties = `
      4lw.commands.whitelist=ruok
      tickTime=2000
      dataDir=/var/lib/zookeeper/data
      dataLogDir=/var/lib/zookeeper/log
      clientPort=2181
    `
	functionalPodzookeeperLog4JProperties = `
      log4j.rootLogger=INFO, stdout
      log4j.appender.stdout=org.apache.log4j.ConsoleAppender
      log4j.appender.stdout.layout=org.apache.log4j.PatternLayout
      log4j.appender.stdout.layout.ConversionPattern=[%d] %p %m (%c)%n

      # Suppress connection log messages, three lines per livenessProbe execution
      log4j.logger.org.apache.zookeeper.server.NIOServerCnxnFactory=WARN
      log4j.logger.org.apache.zookeeper.server.NIOServerCnxn=WARN
    `
)
