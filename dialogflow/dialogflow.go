package dialogflow

import (
	"context"
	"fmt"
	"strconv"

	dflib "cloud.google.com/go/dialogflow/apiv2"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/pkg/errors"
	"google.golang.org/api/option"
	dflibpb "google.golang.org/genproto/googleapis/cloud/dialogflow/v2"
)

type (
	// DialogFlow config
	DFConfig struct {
		ProjectID    string
		JSONFilePath string
		Lang         string
		Timezone     string
	}
	// DialogflowProcessor has all the information for connecting with Dialogflow
	DFProcessor struct {
		ProjectID        string
		AuthJSONFilePath string
		Lang             string
		TimeZone         string
		SessionClient    *dflib.SessionsClient
		Ctx              context.Context
	}
	NLPResponse struct {
		Intent     string            `json:"intent"`
		Confidence float32           `json:"confidence"`
		Entities   map[string]string `json:"entities"`
		Answer     string            `json:"answer"`
		Contextes  map[string]string `json:"contextes"`
	}
	Routes struct {
		Street   string
		Building string
		House    string
		Letter   string
	}
)

func InitAgent(conf DFConfig) (DFProcessor, error) {
	var dp DFProcessor
	dp.ProjectID = conf.ProjectID
	dp.AuthJSONFilePath = conf.JSONFilePath
	dp.Lang = conf.Lang
	dp.TimeZone = conf.Timezone

	dp.Ctx = context.Background()
	sessionClient, err := dflib.NewSessionsClient(dp.Ctx, option.WithCredentialsFile(dp.AuthJSONFilePath))
	if err != nil {
		return DFProcessor{}, err
	}
	dp.SessionClient = sessionClient

	return dp, nil
}

func (dfp *DFProcessor) DetectIntentText(text, sessionID string, dfContext ...string) (NLPResponse, error) {
	var nresp NLPResponse

	var cntxObj dflibpb.Context
	request := dflibpb.DetectIntentRequest{
		Session: fmt.Sprintf("projects/%s/agent/sessions/%s", dfp.ProjectID, string(sessionID)),
		QueryInput: &dflibpb.QueryInput{
			Input: &dflibpb.QueryInput_Text{
				Text: &dflibpb.TextInput{
					Text:         text,
					LanguageCode: dfp.Lang,
				},
			},
		},
		QueryParams: &dflibpb.QueryParameters{
			TimeZone: dfp.TimeZone,
		},
	}

	//добавляем контекст если он есть
	if len(dfContext) > 0 {
		cntxObj = dflibpb.Context{
			Name:          fmt.Sprintf("projects/%s/agent/sessions/%s/contexts/%s", dfp.ProjectID, string(sessionID), dfContext[0]),
			LifespanCount: 1,
		}
		request.QueryParams.Contexts = append(request.QueryParams.Contexts, &cntxObj)
		request.QueryParams.ResetContexts = true
	}

	resp, err := dfp.SessionClient.DetectIntent(dfp.Ctx, &request)
	if err != nil {
		return nresp, errors.Wrap(err, "Error in communication with Dialogflow")
	}

	queryResult := resp.GetQueryResult()
	if queryResult.Intent != nil {
		nresp.Intent = queryResult.Intent.DisplayName
		nresp.Confidence = float32(queryResult.IntentDetectionConfidence)
	}

	nresp.Contextes = make(map[string]string)
	for _, val := range queryResult.OutputContexts {
		fi := val.Parameters.GetFields()
		for k, v := range fi {
			kind := v.GetKind()
			switch kind.(type) {
			case *structpb.Value_StringValue:
				nresp.Contextes[k] = v.GetStringValue()
			case *structpb.Value_NumberValue:
				flstring := strconv.FormatFloat(v.GetNumberValue(), 'f', 6, 64)
				nresp.Contextes[k] = flstring
			}
		}
	}

	nresp.Entities = make(map[string]string)

	params := queryResult.Parameters.GetFields()
	if len(params) > 0 {
		for paramName, p := range params {
			//fmt.Printf("Param %s: %s (%s)\n", paramName, p.GetStringValue(), p.String())
			extractedValue := extractDialogflowEntities(p)
			nresp.Entities[paramName] = extractedValue
		}
	}

	nresp.Answer = queryResult.GetFulfillmentText()
	return nresp, nil
}

func extractDialogflowEntities(p *structpb.Value) (extractedEntity string) {
	kind := p.GetKind()
	switch kind.(type) {
	case *structpb.Value_StringValue:
		return p.GetStringValue()
	case *structpb.Value_NumberValue:
		return strconv.FormatFloat(p.GetNumberValue(), 'f', 6, 64)
	case *structpb.Value_BoolValue:
		return strconv.FormatBool(p.GetBoolValue())
	case *structpb.Value_StructValue:
		s := p.GetStructValue()
		fields := s.GetFields()
		extractedEntity = ""
		for key, value := range fields {
			if key == "amount" {
				extractedEntity = fmt.Sprintf("%s%s", extractedEntity, strconv.FormatFloat(value.GetNumberValue(), 'f', 6, 64))
			}
			if key == "unit" {
				extractedEntity = fmt.Sprintf("%s%s", extractedEntity, value.GetStringValue())
			}
			if key == "date_time" {
				extractedEntity = fmt.Sprintf("%s%s", extractedEntity, value.GetStringValue())
			}
			// @TODO: Other entity types can be added here
		}
		return extractedEntity
	case *structpb.Value_ListValue:
		list := p.GetListValue()
		if len(list.GetValues()) > 1 {
			// @TODO: Extract more values
		}
		extractedEntity = extractDialogflowEntities(list.GetValues()[0])
		return extractedEntity
	default:
		return ""
	}
}
