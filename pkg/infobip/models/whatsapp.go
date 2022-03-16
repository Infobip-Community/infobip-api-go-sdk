package models

import (
	"bytes"
	"fmt"
	"time"
	"unicode"

	"github.com/go-playground/validator/v10"
	"mvdan.cc/xurls/v2"
)

const maxInteractiveListRows = 10

func setupWhatsAppValidations() {
	if validate == nil {
		validate = validator.New()
	}
	validate.RegisterStructValidation(templateCreateValidation, TemplateCreate{})
	validate.RegisterStructValidation(templateCreateButtonValidation, TemplateButton{})
	validate.RegisterStructValidation(templateMsgValidation, TemplateMsg{})
	validate.RegisterStructValidation(templateMsgButtonValidation, TemplateMsgButton{})
	validate.RegisterStructValidation(textMsgValidation, TextMsg{})
	validate.RegisterStructValidation(contactValidation, Contact{})
	validate.RegisterStructValidation(interactiveButtonsMsgValidation, InteractiveButtonsMsg{})
	validate.RegisterStructValidation(interactiveListMsgValidation, InteractiveListMsg{})
	validate.RegisterStructValidation(multiproductMsgValidation, InteractiveMultiproductMsg{})
}

type BulkMsgResponse struct {
	Messages []MsgResponse `json:"messages"`
	BulkID   string        `json:"bulkId"`
}

type MsgResponse struct {
	To           string `json:"to"`
	MessageCount int32  `json:"messageCount"`
	MessageID    string `json:"messageId"`
	Status       Status `json:"status"`
}

type Status struct {
	GroupID     int32  `json:"groupId"`
	GroupName   string `json:"groupName"`
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

type TemplatesResponse struct {
	Templates []TemplateResponse `json:"templates"`
}

type TemplateResponse struct {
	ID                string            `json:"ID"`
	BusinessAccountID int64             `json:"businessAccountID"`
	Name              string            `json:"name"`
	Language          string            `json:"language"`
	Status            string            `json:"status"`
	Category          string            `json:"category"`
	Structure         TemplateStructure `json:"structure"`
}

type TemplateStructure struct {
	Header  *TemplateHeader  `json:"header,omitempty"`
	Body    string           `json:"body" validate:"required"`
	Footer  string           `json:"footer,omitempty" validate:"lte=60"`
	Buttons []TemplateButton `json:"buttons,omitempty" validate:"omitempty,min=1,max=3,dive"`
	Type    string           `json:"type,omitempty" validate:"oneof=TEXT MEDIA UNSUPPORTED"`
}

type TemplateHeader struct {
	Format string `json:"format,omitempty" validate:"oneof=TEXT IMAGE VIDEO DOCUMENT LOCATION"`
	Text   string `json:"text" validate:"lte=60"`
}

type TemplateButton struct {
	Type        string `json:"type" validate:"required,oneof=PHONE_NUMBER URL QUICK_REPLY"`
	Text        string `json:"text" validate:"required,lte=200"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	URL         string `json:"url,omitempty" validate:"omitempty,url"`
}

type TemplateCreate struct {
	Name      string            `json:"name" validate:"required"`
	Language  string            `json:"language" validate:"required"`
	Category  string            `json:"category" validate:"required"`
	Structure TemplateStructure `json:"structure" validate:"required"`
}

func (t *TemplateCreate) Validate() error {
	return validate.Struct(t)
}

func (t *TemplateCreate) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func templateCreateValidation(sl validator.StructLevel) {
	template, _ := sl.Current().Interface().(TemplateCreate)
	validateTemplateName(sl, template)
	validateTemplateLanguage(sl, template)
	validateTemplateCategory(sl, template)
	validateTemplateHeader(sl, template)
	validateTemplateButtons(sl, template)
}

func validateTemplateName(sl validator.StructLevel, template TemplateCreate) {
	if !isSnakeCase(template.Name) {
		sl.ReportError(template.Name, "name", "Name", "namenotsnakecase", "")
	}
}

func validateTemplateLanguage(sl validator.StructLevel, template TemplateCreate) {
	switch template.Language {
	case "af", "sq", "ar", "az", "bn", "bg", "ca", "zh_CN", "zh_HK", "zh_TW", "hr", "cs", "da", "nl", "en", "en_GB",
		"en_US", "et", "fil", "fi", "fr", "de", "el", "gu", "ha", "he", "hi", "hu", "id", "ga", "it", "ja", "kn", "kk",
		"ko", "lo", "lv", "lt", "mk", "ms", "ml", "mr", "nb", "fa", "pl", "pt_BR", "pt_PT", "pa", "ro", "ru", "sr",
		"sk", "sl", "es", "es_AR", "es_ES", "es_MX", "sw", "sv", "ta", "te", "th", "tr", "uk", "ur", "uz", "vi",
		"unknown":
	default:
		sl.ReportError(template.Language, "language", "Language", "invalidlanguage", "")
	}
}

func validateTemplateCategory(sl validator.StructLevel, template TemplateCreate) {
	switch template.Category {
	case "ACCOUNT_UPDATE", "PAYMENT_UPDATE", "PERSONAL_FINANCE_UPDATE", "SHIPPING_UPDATE",
		"RESERVATION_UPDATE", "ISSUE_RESOLUTION", "APPOINTMENT_UPDATE", "TRANSPORTATION_UPDATE",
		"TICKET_UPDATE", "ALERT_UPDATE", "AUTO_REPLY":
	default:
		sl.ReportError(template.Category, "category", "Category", "invalidcategory", "")
	}
}

func validateTemplateHeader(sl validator.StructLevel, template TemplateCreate) {
	header := template.Structure.Header
	if header != nil && header.Format == "TEXT" && header.Text == "" {
		sl.ReportError(header.Text, "text", "Text", "missingtext", "")
	}
}

func validateTemplateButtons(sl validator.StructLevel, template TemplateCreate) {
	types := map[string]int{"QUICK_REPLY": 0, "PHONE_NUMBER": 0, "URL": 0}
	for _, button := range template.Structure.Buttons {
		types[button.Type]++
	}

	if types["QUICK_REPLY"] > 0 && (types["URL"] > 0 || types["PHONE_NUMBER"] > 0) {
		sl.ReportError(template.Structure.Buttons, "buttons", "Buttons", "mixedquickreplyactiontypes", "")
	}
	if types["URL"] > 1 || types["PHONE_NUMBER"] > 1 {
		sl.ReportError(template.Structure.Buttons, "buttons", "Buttons", "multiplesameactiontypes", "")
	}
}

func templateCreateButtonValidation(sl validator.StructLevel) {
	button, _ := sl.Current().Interface().(TemplateButton)
	switch button.Type {
	case "PHONE_NUMBER":
		if button.PhoneNumber == "" {
			sl.ReportError(button.PhoneNumber, "phoneNumber", "PhoneNumber", "required", "")
		}
	case "URL":
		if button.URL == "" {
			sl.ReportError(button.URL, "url", "URL", "required", "")
		}
	}
}

type MsgCommon struct {
	From         string `json:"from" validate:"required,lte=24"`
	To           string `json:"to" validate:"required,lte=24"`
	MessageID    string `json:"messageId,omitempty" validate:"lte=50"`
	CallbackData string `json:"callbackData,omitempty" validate:"lte=4000"`
	NotifyURL    string `json:"notifyUrl,omitempty" validate:"omitempty,url,lte=2048"`
}

type TemplateMsgs struct {
	Messages []TemplateMsg `json:"messages" validate:"required,min=1,dive"`
	BulkID   string        `json:"bulkId,omitempty" validate:"lte=100"`
}

func (t *TemplateMsgs) Validate() error {
	return validate.Struct(t)
}

func (t *TemplateMsgs) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func templateMsgValidation(sl validator.StructLevel) {
	msg, _ := sl.Current().Interface().(TemplateMsg)
	validateTemplateMsgName(sl, msg)
	validateTemplateMsgHeader(sl, msg)
	validateTemplateButtonLength(sl, msg.Content.TemplateData)
	validateTemplateButtonTypes(sl, msg.Content.TemplateData)
}

func validateTemplateMsgName(sl validator.StructLevel, msg TemplateMsg) {
	if !isSnakeCase(msg.Content.TemplateName) {
		sl.ReportError(msg.Content.TemplateName, "templateName", "TemplateName", "templatenamenotsnakecase", "")
	}
}

func isSnakeCase(s string) bool {
	for _, r := range s {
		if !unicode.IsLower(r) && r != '_' {
			return false
		}
	}
	return true
}

func validateTemplateMsgHeader(sl validator.StructLevel, msg TemplateMsg) {
	header := msg.Content.TemplateData.Header
	if header == nil {
		return
	}

	switch header.Type {
	case "TEXT":
		if header.Placeholder == "" {
			sl.ReportError(header.Placeholder, "placholder", "Placeholder", "missingplaceholder", "")
		}
	case "DOCUMENT":
		if header.MediaURL == "" {
			sl.ReportError(header.MediaURL, "mediaUrl", "MediaURL", "missingmediaurl", "")
		}
		if header.Filename == "" {
			sl.ReportError(header.Filename, "filename", "Filename", "missingfilename", "")
		}
	case "VIDEO", "IMAGE":
		if header.MediaURL == "" {
			sl.ReportError(header.MediaURL, "mediaUrl", "MediaURL", "missingmediaurl", "")
		}
	case "LOCATION":
		if header.Latitude == nil {
			sl.ReportError(header.Latitude, "latitude", "Latitude", "missinglatitude", "")
		}
		if header.Longitude == nil {
			sl.ReportError(header.Longitude, "longitude", "Longitude", "missinglongitude", "")
		}
	}
}

func validateTemplateButtonLength(sl validator.StructLevel, templateData TemplateData) {
	if len(templateData.Buttons) > 1 && templateData.Buttons[0].Type == "URL" {
		sl.ReportError(templateData.Buttons, "buttons", "Buttons", "dynamicurlcountoverone", "")
	}
}

func validateTemplateButtonTypes(sl validator.StructLevel, templateData TemplateData) {
	types := map[string]int{"QUICK_REPLY": 0, "URL": 0}
	for _, button := range templateData.Buttons {
		types[button.Type]++
	}
	if types["QUICK_REPLY"] > 0 && types["URL"] > 0 {
		sl.ReportError(templateData.Buttons, "buttons", "Buttons", "bothreplyandurlpresent", "")
	}
}

func templateMsgButtonValidation(sl validator.StructLevel) {
	button, _ := sl.Current().Interface().(TemplateMsgButton)
	if button.Type == "QUICK_REPLY" && len(button.Parameter) > 128 {
		sl.ReportError(button.Parameter, "parameter", "Parameter", "parametertoolong", "")
	}
}

type TemplateMsg struct {
	MsgCommon
	Content     TemplateMsgContent `json:"content" validate:"required"`
	SMSFailover *SMSFailover       `json:"smsFailover,omitempty"`
}

type TemplateMsgContent struct {
	TemplateName string       `json:"templateName" validate:"required,lte=512"`
	TemplateData TemplateData `json:"templateData" validate:"required"`
	Language     string       `json:"language" validate:"required"`
}

type TemplateData struct {
	Body    TemplateBody        `json:"body" validate:"required"`
	Header  *TemplateMsgHeader  `json:"header,omitempty"`
	Buttons []TemplateMsgButton `json:"buttons,omitempty" validate:"omitempty,max=3,dive"`
}

type TemplateBody struct {
	Placeholders []string `json:"placeholders" validate:"required,dive,gte=1"`
}

type TemplateMsgHeader struct {
	Type        string   `json:"type" validate:"required,oneof=TEXT DOCUMENT IMAGE VIDEO LOCATION"`
	Placeholder string   `json:"placeholder,omitempty"`
	MediaURL    string   `json:"mediaUrl,omitempty" validate:"omitempty,url,lte=2048"`
	Filename    string   `json:"filename,omitempty" validate:"lte=240"`
	Latitude    *float32 `json:"latitude,omitempty" validate:"omitempty,latitude"`
	Longitude   *float32 `json:"longitude,omitempty" validate:"omitempty,longitude"`
}

type TemplateMsgButton struct {
	Type      string `json:"type" validate:"required,oneof=QUICK_REPLY URL"`
	Parameter string `json:"parameter" validate:"required"`
}

type SMSFailover struct {
	From string `json:"from" validate:"required,lte=24"`
	Text string `json:"text" validate:"required,lte=4096"`
}

type TextMsg struct {
	MsgCommon
	Content TextContent `json:"content" validate:"required"`
}

func (t *TextMsg) Validate() error {
	return validate.Struct(t)
}

func (t *TextMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func textMsgValidation(sl validator.StructLevel) {
	msg, _ := sl.Current().Interface().(TextMsg)
	previewURLValidation(sl, msg)
}

type TextContent struct {
	Text       string `json:"text" validate:"required,gte=1,lte=4096"`
	PreviewURL bool   `json:"previewURL,omitempty"`
}

func previewURLValidation(sl validator.StructLevel, msg TextMsg) {
	content := msg.Content
	containsURL := xurls.Relaxed().FindString(content.Text)
	if content.PreviewURL && containsURL == "" {
		sl.ReportError(msg.Content.Text, "text", "Text", "missingurlintext", "")
	}
}

type DocumentMsg struct {
	MsgCommon
	Content DocumentContent `json:"content" validate:"required"`
}

func (t *DocumentMsg) Validate() error {
	return validate.Struct(t)
}

func (t *DocumentMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type DocumentContent struct {
	MediaURL string `json:"mediaUrl" validate:"required,url,lte=2048"`
	Caption  string `json:"caption,omitempty" validate:"lte=3000"`
	Filename string `json:"filename,omitempty" validate:"lte=240"`
}

type ImageMsg struct {
	MsgCommon
	Content ImageContent `json:"content" validate:"required"`
}

func (t *ImageMsg) Validate() error {
	return validate.Struct(t)
}

func (t *ImageMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type ImageContent struct {
	MediaURL string `json:"mediaUrl" validate:"required,url,lte=2048"`
	Caption  string `json:"caption,omitempty" validate:"lte=3000"`
}

type AudioMsg struct {
	MsgCommon
	Content AudioContent `json:"content" validate:"required"`
}

func (t *AudioMsg) Validate() error {
	return validate.Struct(t)
}

func (t *AudioMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type AudioContent struct {
	MediaURL string `json:"mediaUrl" validate:"required,url,lte=2048"`
}

type VideoMsg struct {
	MsgCommon
	Content VideoContent `json:"content" validate:"required"`
}

func (t *VideoMsg) Validate() error {
	return validate.Struct(t)
}

func (t *VideoMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type VideoContent struct {
	MediaURL string `json:"mediaUrl" validate:"required,url,lte=2048"`
	Caption  string `json:"caption,omitempty" validate:"lte=3000"`
}

type StickerMsg struct {
	MsgCommon
	Content StickerContent `json:"content" validate:"required"`
}

func (t *StickerMsg) Validate() error {
	return validate.Struct(t)
}

func (t *StickerMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type StickerContent struct {
	MediaURL string `json:"mediaUrl" validate:"required,url,lte=2048"`
}

type LocationMsg struct {
	MsgCommon
	Content LocationContent `json:"content" validate:"required"`
}

func (t *LocationMsg) Validate() error {
	return validate.Struct(t)
}

func (t *LocationMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type LocationContent struct {
	Latitude  *float32 `json:"latitude" validate:"required,latitude"`
	Longitude *float32 `json:"longitude" validate:"required,longitude"`
	Name      string   `json:"name" validate:"lte=1000"`
	Address   string   `json:"address" validate:"lte=1000"`
}

type ContactMsg struct {
	MsgCommon
	Content ContactContent `json:"content" validate:"required"`
}

func (t *ContactMsg) Validate() error {
	return validate.Struct(t)
}

func (t *ContactMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func contactValidation(sl validator.StructLevel) {
	contact, _ := sl.Current().Interface().(Contact)
	if contact.Birthday == "" {
		return
	}
	_, err := time.Parse("2006-01-02", contact.Birthday)
	if err != nil {
		sl.ReportError(contact.Birthday, "birthday", "Contact", "invalidbirthdayformat", "")
	}
}

type ContactContent struct {
	Contacts []Contact `json:"contacts" validate:"required,dive"`
}

type Contact struct {
	Addresses []ContactAddress `json:"addresses,omitempty" validate:"omitempty,dive"`
	Birthday  string           `json:"birthday,omitempty"`
	Emails    []ContactEmail   `json:"emails,omitempty" validate:"omitempty,dive"`
	Name      ContactName      `json:"name" validate:"required"`
	Org       ContactOrg       `json:"org,omitempty"`
	Phones    []ContactPhone   `json:"phones,omitempty" validate:"omitempty,dive"`
	Urls      []ContactURL     `json:"urls,omitempty" validate:"omitempty,dive"`
}

type ContactAddress struct {
	Street      string `json:"street,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	Zip         string `json:"zip,omitempty"`
	Country     string `json:"country,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
	Type        string `json:"type,omitempty" validate:"omitempty,oneof=HOME WORK"`
}

type ContactEmail struct {
	Email string `json:"email,omitempty" validate:"omitempty,email"`
	Type  string `json:"type,omitempty" validate:"omitempty,oneof=HOME WORK"`
}

type ContactName struct {
	FirstName     string `json:"firstName" validate:"required"`
	LastName      string `json:"lastName,omitempty"`
	MiddleName    string `json:"middleName,omitempty"`
	NameSuffix    string `json:"nameSuffix,omitempty"`
	NamePrefix    string `json:"namePrefix,omitempty"`
	FormattedName string `json:"formattedName" validate:"required"`
}

type ContactOrg struct {
	Company    string `json:"company,omitempty"`
	Department string `json:"department,omitempty"`
	Title      string `json:"title,omitempty"`
}

type ContactPhone struct {
	Phone string `json:"phone,omitempty"`
	Type  string `json:"type,omitempty" validate:"omitempty,oneof=CELL MAIN IPHONE HOME WORK"`
	WaID  string `json:"waId,omitempty"`
}

type ContactURL struct {
	URL  string `json:"url,omitempty" validate:"omitempty,url"`
	Type string `json:"type,omitempty" validate:"omitempty,oneof=HOME WORK"`
}

type InteractiveButtonsMsg struct {
	MsgCommon
	Content InteractiveButtonsContent `json:"content" validate:"required"`
}

func (t *InteractiveButtonsMsg) Validate() error {
	return validate.Struct(t)
}

func (t *InteractiveButtonsMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func interactiveButtonsMsgValidation(sl validator.StructLevel) {
	msg, _ := sl.Current().Interface().(InteractiveButtonsMsg)
	validateInteractiveButtonsHeader(sl, msg)
}

func validateInteractiveButtonsHeader(sl validator.StructLevel, msg InteractiveButtonsMsg) {
	header := msg.Content.Header
	if header == nil {
		return
	}

	switch header.Type {
	case "TEXT":
		if header.Text == "" {
			sl.ReportError(msg.Content.Header.Text, "text", "Text", "missingtext", "")
		}
	case "VIDEO", "IMAGE", "DOCUMENT":
		if header.MediaURL == "" {
			sl.ReportError(msg.Content.Header.MediaURL, "mediaUrl", "MediaURL", "missingmediaurl", "")
		}
	}
}

type InteractiveButtonsContent struct {
	Body   InteractiveButtonsBody    `json:"body" validate:"required"`
	Action InteractiveButtons        `json:"action" validate:"required"`
	Header *InteractiveButtonsHeader `json:"header,omitempty" validate:"omitempty"`
	Footer *InteractiveButtonsFooter `json:"footer,omitempty"`
}

type InteractiveButtonsBody struct {
	Text string `json:"text" validate:"required,lte=1024"`
}

type InteractiveButtons struct {
	Buttons []InteractiveButton `json:"buttons" validate:"required,min=1,max=3,dive"`
}

type InteractiveButton struct {
	Type  string `json:"type" validate:"required,oneof=REPLY"`
	ID    string `json:"id" validate:"required,lte=256"`
	Title string `json:"title" validate:"required,lte=20"`
}

type InteractiveButtonsHeader struct {
	Type     string `json:"type" validate:"required,oneof=TEXT VIDEO IMAGE DOCUMENT"`
	Text     string `json:"text,omitempty" validate:"lte=60"`
	MediaURL string `json:"mediaUrl,omitempty" validate:"omitempty,url,lte=2048"`
	Filename string `json:"filename,omitempty" validate:"lte=240"`
}

type InteractiveButtonsFooter struct {
	Text string `json:"text" validate:"required,lte=60"`
}

type InteractiveListMsg struct {
	MsgCommon
	Content InteractiveListContent `json:"content" validate:"required"`
}

func (t *InteractiveListMsg) Validate() error {
	return validate.Struct(t)
}

func (t *InteractiveListMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func interactiveListMsgValidation(sl validator.StructLevel) {
	msg, _ := sl.Current().Interface().(InteractiveListMsg)
	validateListSectionRowCount(sl, msg)
	validateListDuplicateRows(sl, msg)
	validateListSectionTitles(sl, msg)
}

func validateListSectionRowCount(sl validator.StructLevel, msg InteractiveListMsg) {
	var rowCount int
	sections := msg.Content.Action.Sections

	for _, section := range sections {
		rowCount += len(section.Rows)
		if rowCount > maxInteractiveListRows {
			sl.ReportError(
				msg.Content.Action.Sections,
				"sections",
				"Sections",
				"rowcountovermax",
				"",
			)
		}
	}
}

func validateListDuplicateRows(sl validator.StructLevel, msg InteractiveListMsg) {
	rowIDs := make(map[string]int)
	sections := msg.Content.Action.Sections

	for _, section := range sections {
		for _, row := range section.Rows {
			rowIDs[row.ID]++
			if rowIDs[row.ID] > 1 {
				sl.ReportError(
					msg.Content.Action.Sections,
					"sections",
					"Sections",
					fmt.Sprintf("duplicaterowID%s", row.ID),
					"",
				)
			}
		}
	}
}

func validateListSectionTitles(sl validator.StructLevel, msg InteractiveListMsg) {
	sections := msg.Content.Action.Sections

	if len(sections) > 1 {
		for _, section := range sections {
			if section.Title == "" {
				sl.ReportError(
					msg.Content.Action.Sections,
					"sections",
					"Sections",
					"missingtitlemultiplesections",
					"",
				)
			}
		}
	}
}

type InteractiveListContent struct {
	Body   InteractiveListBody    `json:"body" validate:"required"`
	Action InteractiveListAction  `json:"action" validate:"required"`
	Header *InteractiveListHeader `json:"header,omitempty" validate:"omitempty"`
	Footer *InteractiveListFooter `json:"footer,omitempty"`
}

type InteractiveListBody struct {
	Text string `json:"text" validate:"required,lte=1024"`
}

type InteractiveListAction struct {
	Title    string                   `json:"title" validate:"required,lte=20"`
	Sections []InteractiveListSection `json:"sections" validate:"required,min=1,max=10,dive"`
}

type InteractiveListSection struct {
	Title string       `json:"title,omitempty" validate:"lte=24"`
	Rows  []SectionRow `json:"rows" validate:"required,min=1,max=10,dive"`
}

type SectionRow struct {
	ID          string `json:"id" validate:"required,lte=200"`
	Title       string `json:"title" validate:"required,lte=24"`
	Description string `json:"description,omitempty" validate:"lte=72"`
}

type InteractiveListHeader struct {
	Type string `json:"type" validate:"required,oneof=TEXT"`
	Text string `json:"text" validate:"required,lte=60"`
}

type InteractiveListFooter struct {
	Text string `json:"text" validate:"required,lte=60"`
}

type InteractiveProductMsg struct {
	MsgCommon
	Content InteractiveProductContent `json:"content" validate:"required"`
}

func (t *InteractiveProductMsg) Validate() error {
	return validate.Struct(t)
}

func (t *InteractiveProductMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

type InteractiveProductContent struct {
	Action InteractiveProductAction  `json:"action" validate:"required"`
	Body   *InteractiveProductBody   `json:"body,omitempty"`
	Footer *InteractiveProductFooter `json:"footer,omitempty"`
}

type InteractiveProductAction struct {
	CatalogID         string `json:"catalogId" validate:"required"`
	ProductRetailerID string `json:"productRetailerId" validate:"required"`
}

type InteractiveProductBody struct {
	Text string `json:"text" validate:"required,lte=1024"`
}

type InteractiveProductFooter struct {
	Text string `json:"text" validate:"required,lte=60"`
}

type InteractiveMultiproductMsg struct {
	MsgCommon
	Content InteractiveMultiproductContent `json:"content" validate:"required"`
}

func (t *InteractiveMultiproductMsg) Validate() error {
	return validate.Struct(t)
}

func (t *InteractiveMultiproductMsg) Marshal() (*bytes.Buffer, error) {
	return marshalJSON(t)
}

func multiproductMsgValidation(sl validator.StructLevel) {
	msg, _ := sl.Current().Interface().(InteractiveMultiproductMsg)
	validateMultiproductSectionTitles(sl, msg)
}

func validateMultiproductSectionTitles(sl validator.StructLevel, msg InteractiveMultiproductMsg) {
	sections := msg.Content.Action.Sections
	if len(sections) > 1 {
		for _, section := range sections {
			if section.Title == "" {
				sl.ReportError(
					msg.Content.Action.Sections,
					"sections",
					"Sections",
					"missingtitlemultiplesections",
					"",
				)
			}
		}
	}
}

type InteractiveMultiproductContent struct {
	Header InteractiveMultiproductHeader  `json:"header" validate:"required"`
	Body   InteractiveMultiproductBody    `json:"body" validate:"required"`
	Action InteractiveMultiproductAction  `json:"action" validate:"required"`
	Footer *InteractiveMultiproductFooter `json:"footer,omitempty"`
}

type InteractiveMultiproductHeader struct {
	Type string `json:"type" validate:"required,oneof=TEXT"`
	Text string `json:"text" validate:"required,lte=60"`
}

type InteractiveMultiproductBody struct {
	Text string `json:"text" validate:"required,lte=1024"`
}

type InteractiveMultiproductAction struct {
	CatalogID string                           `json:"catalogId" validate:"required"`
	Sections  []InteractiveMultiproductSection `json:"sections" validate:"required,min=1,max=10,dive"`
}

type InteractiveMultiproductSection struct {
	Title              string   `json:"title,omitempty" validate:"lte=24"`
	ProductRetailerIDs []string `json:"productRetailerIds" validate:"required,min=1"`
}

type InteractiveMultiproductFooter struct {
	Text string `json:"text" validate:"required,lte=60"`
}
