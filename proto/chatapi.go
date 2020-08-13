// CHATAPI structures
package proto

type WhAppMsg struct {
	ChatId string `json:"chatId"`
	Phone  int    `json:"phone"`
	Body   string `json:"body"`
}

type WhAppMsgResp struct {
	Sent    bool   `json:"sent"`
	Id      string `json:"id"`
	Message string `json:"message"`
}
