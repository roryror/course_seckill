package internal

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

var kafkaConn *kafka.Conn
var kafkaReader *kafka.Reader
var kafkaWriter *kafka.Writer

func InitKafka() {
	config := GlobalConfig.Kafka
	
	// in this demo we only use one broker
	var err error
	kafkaConn, err = kafka.Dial("tcp", config.Brokers[0])
	if err != nil {
		fmt.Printf("Kafka connection failed: %v\n", err)
		return
	}

	// create topic
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             config.Topic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	err = kafkaConn.CreateTopics(topicConfigs...)
	if err != nil {
		fmt.Printf("Create topic failed: %v\n", err)
		// continue, because the topic may already exist
	}

	// wait for topic creation to complete
	time.Sleep(time.Second * 2)

	// initialize writer
	kafkaWriter = kafka.NewWriter(kafka.WriterConfig{
		Brokers:      config.Brokers,
		Topic:        config.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: 1,
		Async:        false,
		MaxAttempts:  config.MaxAttempts,
		BatchTimeout: config.BatchTimeout,
	})

	// initialize reader
	kafkaReader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:         config.Topic,
		GroupID:       config.GroupID,
		MinBytes:      int(config.MinBytes),
		MaxBytes:      int(config.MaxBytes),
		MaxWait:       config.MaxWait,
		StartOffset:   kafka.FirstOffset,
		ReadBackoffMin: config.ReadBackoffMin,
		ReadBackoffMax: config.ReadBackoffMax,
		CommitInterval: config.CommitInterval,
	})
	// start a goroutine(channel buffer) connecting request and kafka
	go processMessages()
	// start a goroutine to handle mysql orders
	go startConsumer()
	// activate reader
	sendMessage(0, 0)
	fmt.Println("Kafka initialized")
}

type orderMessage struct {
	UserID    int
	CourseID  int
}

const (
	batchSize = 300   
	channelSize = 10000
	flushInterval = time.Second / 10	
)

// channel buffer to minimize the delay of http request
// buffer size is 10000
var messageChan = make(chan orderMessage, channelSize)

// process messages from channel buffer
// then send messages to kafka in batch
func processMessages() {
	var messageBatch = make([]orderMessage, 0, batchSize)
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case msg := <-messageChan:
			messageBatch = append(messageBatch, msg)
			
			if len(messageBatch) >= batchSize {
				sendBatchMessages(messageBatch)
				messageBatch = messageBatch[:0]
			}
		// send messages every flushInterval
		case <-ticker.C:
			if len(messageBatch) > 0 {
				sendBatchMessages(messageBatch)
				messageBatch = messageBatch[:0]
			}
		}
	}
}

// send messages to kafka
func sendBatchMessages(messages []orderMessage) {
	if len(messages) == 0 {
		return
	}

	kafkaMessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		value := fmt.Sprintf("%d:%d", msg.UserID, msg.CourseID)
		kafkaMessages[i] = kafka.Message{
			Value: []byte(value),
		}
	}

	err := kafkaWriter.WriteMessages(ctx, kafkaMessages...)
	if err != nil {
		fmt.Printf("batch send message failed: %v\n", err)
		for _, msg := range messages {
			rollbackRedis(msg.CourseID)
		}
		return
	}

	fmt.Printf("batch send message success -> %d\n", len(messages))
}

func sendMessage(uid int, cid int) error {
	if kafkaWriter == nil {
		return fmt.Errorf("kafka writer not initialized")
	}
	if uid == 0 && cid == 0 {
		message := "activate reader"
		return kafkaWriter.WriteMessages(ctx, kafka.Message{Value: []byte(message)})
	}

	message := fmt.Sprintf("%d:%d", uid, cid)	
	for i := range 3 {
		err := kafkaWriter.WriteMessages(ctx, 
			kafka.Message{
				Value: []byte(message),
			},
		)
		
		if err == nil {
			fmt.Println("send order success -> ", message)
			return nil
		}
		
		fmt.Printf("send message attempt %d/3 failed: %v\n", i+1, err)
		time.Sleep(time.Second)
	}
	
	return fmt.Errorf("send message failed")
}

// start consumer to handle kafka messages
// in this demo, we only use one consumer
func startConsumer() {
	parseMessage := func(message string) (int, int) {
		parts := strings.Split(message, ":")
		uid, _ := strconv.Atoi(parts[0])
		cid, _ := strconv.Atoi(parts[1]) 
		return uid, cid
	}
	for {
		msg, err := kafkaReader.ReadMessage(ctx)
		if err != nil {
			if err == io.EOF {
				continue
			}
			fmt.Println	("failed to read message:", err)
		}
		if string(msg.Value) == "activate reader" {
			fmt.Println("Reader activated, ready for seckill!")
			continue
		}
		uid, cid := parseMessage(string(msg.Value))
		createOrder(uid, cid)
	}
}

func CloseKafka() {
	kafkaReader.Close()
	kafkaWriter.Close()
	kafkaConn.Close()
}

