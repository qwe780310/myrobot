package main

import (
	"context"
	"fmt"
	"github.com/robfig/cron"
	"github.com/tencent-connect/botgo"
	"github.com/tencent-connect/botgo/dto"
	"github.com/tencent-connect/botgo/dto/message"
	"github.com/tencent-connect/botgo/event"
	"github.com/tencent-connect/botgo/openapi"
	"github.com/tencent-connect/botgo/token"
	"github.com/tencent-connect/botgo/websocket"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"

	"os"
	"strconv"
	"strings"
	"time"
)

var config Config
var api openapi.OpenAPI
var ctx context.Context
var channelId = "" // 保存子频道的id
var recodeTable = make(map[int]int)
var p = make(map[string]map[int]int)
var flag bool
var guildID string
var userID string
var userIDList []string

//Config 定义了配置文件的结构
type Config struct {
	AppID uint64 `yaml:"appid"` //机器人的appid
	Token string `yaml:"token"` //机器人的token
}

func init() {
	content, err := ioutil.ReadFile("F:\\workspace\\GoWorks\\src\\myproject\\first_server\\test\\config.yaml")
	if err != nil {
		log.Println("读取配置文件出错， err = ", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		log.Println("解析配置文件出错， err = ", err)
		os.Exit(1)
	}
	log.Println(config)
}

func punch(channelID string, messageID string) error {
	flag = false
	day := time.Now().Day()
	content := "请勿重复打卡"
	// 检查是否已经打卡
	// 通配符，任意变量名称
	if recodeTable, ok := p[userID]; ok {
		if _, ok := recodeTable[day]; !ok {
			recodeTable[day] = 1
			content = "打卡成功"
		}
		p[userID] = recodeTable
	} else {
		m := make(map[int]int)
		m[day] = 1
		content = "打卡成功"
		p[userID] = m
	}

	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: content,
	})
	if err != nil {
		return err
	}
	return nil
}

func queryList(channelID string, messageID string) error {
	var content string
	day := time.Now().Day()
	for i := 1; i <= day; i++ {
		recoded := "已打卡"
		recodeTable = p[userID]
		if _, ok := recodeTable[i]; !ok {
			recoded = "未打卡"
		}
		tempDay := strconv.FormatInt(int64(i), 10)
		if i < 10 {
			tempDay = "0" + strconv.FormatInt(int64(i), 10)
		}
		key := "202205" + tempDay
		content = content + "\n" + strconv.FormatInt(int64(i), 10) + "、" + key + ":" + recoded
	}
	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: content,
	})
	if err != nil {
		return err
	}
	return nil
}

func repair(channelID string, messageID string) error {
	flag = true
	currentDay := time.Now().Day()
	start := currentDay
	var content string

	for i := 1; i <= 7 && currentDay > 0; i++ {
		recoded := "已打卡"
		recodeTable = p[userID]
		if _, ok := recodeTable[currentDay]; !ok {
			recoded = "未打卡"
		}
		tempDay := strconv.FormatInt(int64(currentDay), 10)
		if currentDay < 10 {
			tempDay = "0" + strconv.FormatInt(int64(currentDay), 10)
		}
		key := "202205" + tempDay
		content = content + "\n" + strconv.FormatInt(int64(currentDay), 10) + "、" + key + ":" + recoded
		currentDay--
	}
	end := currentDay + 1

	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: content,
	})
	if err != nil {
		return err
	}

	_, err = api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: fmt.Sprintf("请输入 %d - %d选择需要补卡的日期", end, start),
	})
	if err != nil {
		return err
	}
	return nil
}

func updateMap(botMessage string, channelID string, messageID string) error {
	if flag {
		flag = false
		currentDay := time.Now().Day()
		start := currentDay
		end := currentDay - 6

		split := strings.Split(botMessage, " ")
		day, err := strconv.ParseInt(split[1], 10, 64)
		log.Println(start)
		log.Println(end)
		if day > int64(start) || day < int64(end) {
			_, err = api.PostMessage(ctx, channelID, &dto.MessageToCreate{
				MsgID: messageID, Content: "输入补卡日期不合法！",
			})
			return nil
		}
		if err != nil {
			return err
		}
		content := "请勿重复打卡"
		// 检查是否已经打卡
		if _, ok := recodeTable[int(day)]; !ok {
			recodeTable = p[userID]
			recodeTable[int(day)] = 1
			content = "补卡成功"
			return nil
		}
		_, err = api.PostMessage(ctx, channelID, &dto.MessageToCreate{
			MsgID: messageID, Content: content,
		})
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

//func (o *openAPI) PostMessage(ctx context.Context, channelID string, msg *dto.MessageToCreate) (*dto.Message, error) {
//	resp, err := o.request(ctx).
//		SetResult(dto.Message{}).
//		SetPathParam("channel_id", channelID).
//		SetBody(msg).
//		Post(o.getURL(messagesURI))
//	if err != nil {
//		return nil, err
//	}
//
//	return resp.Result().(*dto.Message), nil
//}
//
//func (o *openAPI) request(ctx context.Context) *resty.Request {
//	return o.restyClient.R().SetContext(ctx)
//}

//atMessageEventHandler 处理 @机器人 的消息
func atMessageEventHandler(event *dto.WSPayload, data *dto.WSATMessageData) error {

	channelId = data.ChannelID //当@机器人时，保存ChannelId，主动消息需要channelId才能发送出去
	userID = data.Author.ID
	botMessage := data.Content
	guildID = data.GuildID

	log.Println("botMessage" + botMessage)
	res := message.ParseCommand(botMessage) // 去掉@结构和清除前后空格
	log.Println("cmd = " + res.Cmd + " content = " + res.Content)

	botMessage = strings.TrimRight(botMessage, " ")
	if strings.HasSuffix(botMessage, "打卡") {
		err := punch(data.ChannelID, data.ID)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(botMessage, "查询打卡信息") {
		err := queryList(data.ChannelID, data.ID)
		if err != nil {
			return err
		}
		return nil
	}

	if strings.HasSuffix(botMessage, "补卡") {
		err := repair(data.ChannelID, data.ID)
		if err != nil {
			return err
		}
		return nil
	}

	// 修改打卡状态
	err := updateMap(botMessage, data.ChannelID, data.ID)
	if err != nil {
		return err
	}

	// 如果指令都未被捕获，则说明输入指令有问题
	err = tryException(data.ChannelID, data.ID)
	if err != nil {
		return err
	}

	println(fmt.Sprintf("打卡信息：%#v", recodeTable))

	return nil
}

func tryException(channelID string, messageID string) error {
	_, err := api.PostMessage(ctx, channelID, &dto.MessageToCreate{
		MsgID: messageID, Content: "指令输入非法，请重新输入指令！",
	})
	if err != nil {
		return err
	}
	return nil
}

//registerMsgPush 注册定时器
func registerMsgPush() {

	//channels, err := api.Channels(ctx, guildID)
	currentDay := time.Now().Day()

	var activeMsgPush = func() {
		if channelId != "" {
			//MsgID 为空字符串表示主动消息
			api.PostMessage(ctx, channelId, &dto.MessageToCreate{MsgID: "", Content: "打卡时间到！"})
		}
	}
	var activeMsgPush2 = func() {
		userMap := getUserList()
		fmt.Println(fmt.Sprintf("%v", userMap))
		recodeTable = p[userID]
		content := "未打卡名单：\n"
		for _, member := range userMap {
			recodeTable = p[member.User.ID]
			if channelId != "" && (recodeTable[currentDay] == 0 && !member.User.Bot) {
				content += member.User.Username + "\n"
			}
		}
		_, err := api.PostMessage(ctx, channelId, &dto.MessageToCreate{MsgID: "", Content: content})
		if err != nil {
			log.Println(err)
		}
	}
	timer := cron.New()
	//cron表达式由6部分组成，从左到右分别表示 秒 分 时 日 月 星期
	//*表示任意值  ？表示不确定值，只能用于星期和日
	err := timer.AddFunc("0 0 08 * * ?", activeMsgPush)
	if err != nil {
		return
	}
	err = timer.AddFunc("0 21 23 * * ?", activeMsgPush2)
	if err != nil {
		return
	}
	timer.Start()
}

func getUserList() map[string]*dto.Member {
	userMap := make(map[string]*dto.Member)
	pager := &dto.GuildMembersPager{Limit: "500"}
	members, err := api.GuildMembers(ctx, guildID, pager)
	if err != nil {
		log.Println(err)
	}
	for _, member := range members {
		userMap[member.User.ID] = member
		//append(userMap, member.User.Username)
	}
	return userMap
}

func main() {
	//第二步：生成token，用于校验机器人的身份信息
	token := token.BotToken(config.AppID, config.Token)
	//第三步：获取操作机器人的API对象
	api = botgo.NewOpenAPI(token).WithTimeout(3 * time.Second)
	//获取context
	ctx = context.Background()
	//第四步：获取websocket
	ws, err := api.WS(ctx, nil, "")
	if err != nil {
		log.Fatalln("websocket错误， err = ", err)
		os.Exit(1)
	}
	registerMsgPush()
	var atMessage event.ATMessageEventHandler = atMessageEventHandler

	intent := websocket.RegisterHandlers(atMessage)     // 注册socket消息处理
	botgo.NewSessionManager().Start(ws, token, &intent) // 启动socket监听
}
