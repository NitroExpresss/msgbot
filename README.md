# Управление диалогами

В этом документе кратко описана модель управления сервисов бота

Существует 2 типа обработки команд:

1.Обратка естественного языка. Когда мы получает от пользователя текст,
то отправляем его на распознование в DialogFlow и уже по имени интета
понимаем следующее действие

1.Мы получаем callback что является явной командой

Полученная команда (имя интента или callback) является статусом в который стремится диалог
Управлением перехода из статуса в статус занимаемся State Machine именно она переводит из одного статуса
в другой, а ее callback-и сохраняют изменения статуса и делают дополнительные обработки

Всте стаусы прописаны как константы в `proto.States`

Контекст разгоовра храниться в БД и перед каждой обработкой (сообщением) мы идем в базу и забираем его статус

*DialogFlow* - на входе принимает контекст, которые является текущим статусом заказа. Имя интента - статус к которому
стремиться диалог

## Создание новой ветки разговора

На примере создания новой ветки заказа еды

1. Создать новый статус в `proto.States`. В моем случае я создал Food.Order.Create и прописал значение этого статуса в 
функции `initState()`, именно это значение примет статус заказа в БД `States.Food.Order.Create = "food_order_create"`
1. Прописать этот новый статус в StateMachine - `handler\init_state_machine.go InitOrderStateFSM()`
(Src: - из какого статуса можно в него попасть). После прописания диалог сможет автоматически переключиться во вновь созданный
статус если мы пропишем CallBack или создадим интент с таким именем 
1. Теперь прописываем кнопки что бы можно было в этот статус перейти. В моем случае это `welcome` статус который прописан
хардоком в начале функции получения ответа `h.getTextAnswer(msg)`. Если кнопка нужна для какого то статуса то ее нужно
прописать в функции `getAnswerButtons(msg *models.ChatMsgFull)` в файле `buttons_handler.go`. Нам нужно прописать 
тот статус в котором должна появится кнопка вызывающая наш новый стейт. Стейт задается значением `Data` структуры `MsgButtons`
в хелперах. 
1.Для получения ответа который придет при вызове этого callback-а, нужно его прописать в файле `proto/answers.go` в функции 
инициализации ответов `initAnswers()`. В мапе `Answers.Intents[Constant] []string` ключем является обявленный на 1 этапе
статус типа данных `Constant` а значением массив из нескольких вариантов ответа.
```
var FoodOrderCreate = []string{
	"Мне жаль 😔 но этот раздел еще в разработке. \nМы планируем его запустить 15 августа. Я напишу, как он будет готов.\n\nПока можно заказать такси.",
	"эээээ 😶 заказ еды еще не включили.\nОн запуститься в начале августа, я пришлю уведомление как он будет готов.\nА пока можно заказать такси)",
}
...
//создание заказа еды
for _, v := range FoodOrderCreate {
    Answers.Intents[States.Food.Order.Create] = append(Answers.Intents[States.Food.Order.Create], v)
}
```

теперь функция `proto.GetIntentText(proto.Constant(msg.State))` будет возращать наш текст
1.Что бы создать кнопки для ответа бежим в функцию `handler/button_handler.go - getAnswerButtons(msg)`. 
и прописываем соответствующий 
```
	case proto.States.Food.Order.Create.S():
		return proto.ButtonsSet{
			DisplayLocation: proto.Buttons.Display.Inline,
			Buttons: []proto.MsgKeyboardRows{
				{
					MsgButtons: []proto.MsgButton{{
							Text: proto.Buttons.Menu.CallTaxi.T(),
							Data: proto.Buttons.Menu.CallTaxi.D(),
						}}}}}
```

