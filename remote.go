/* Remote Selenium client implementation.

See http://code.google.com/p/selenium/wiki/JsonWireProtocol for wire protocol.
*/

package selenium

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"
)

var Log = log.New(os.Stderr, "[selenium] ", log.Ltime|log.Lmicroseconds)
var Trace bool

/* Errors returned by Selenium server. */
var errorCodes = map[int]string{
	7:  "no such element",
	8:  "no such frame",
	9:  "unknown command",
	10: "stale element reference",
	11: "element not visible",
	12: "invalid element state",
	13: "unknown error",
	15: "element is not selectable",
	17: "javascript error",
	19: "xpath lookup error",
	21: "timeout",
	23: "no such window",
	24: "invalid cookie domain",
	25: "unable to set cookie",
	26: "unexpected alert open",
	27: "no alert open",
	28: "script timeout",
	29: "invalid element coordinates",
	32: "invalid selector",
}

const (
	SUCCESS         = 0
	defaultExecutor = "http://127.0.0.1:4444/wd/hub"
	jsonMIMEType    = "application/json"
)

type remoteWebDriver struct {
	id, executor string
	capabilities Capabilities
	// FIXME
	// profile             BrowserProfile
	ctx context.Context

	haveQuitMu sync.Mutex
	haveQuit   bool
}

func (wd *remoteWebDriver) SetContext(ctx context.Context) {
	wd.ctx = ctx
}

func (wd *remoteWebDriver) url(template string, args ...interface{}) string {
	path := fmt.Sprintf(template, args...)
	return wd.executor + path
}

func (wd *remoteWebDriver) send(ctx context.Context, method, url string, data []byte) (r *reply, err error) {
	var buf []byte
	if buf, err = wd.execute(ctx, method, url, data); err == nil {
		if len(buf) > 0 {
			err = json.Unmarshal(buf, &r)
		}
	}
	return
}

// VoidExecute ...
func (wd *remoteWebDriver) VoidExecute(ctx context.Context, url string, params interface{}) error {
	return wd.voidCommand(ctx, url, params)
}

// ErrCanceled is returned when the context is cancelled.
var ErrCanceled = errors.New("cancelled")

func (wd *remoteWebDriver) execute(ctx context.Context, method, url string, data []byte) (buf []byte, err error) {
	select {
	case <-wd.ctx.Done():
		err = ErrCanceled
		_ = wd.Quit(context.Background())
		return
	default:
	}
	defer func() {
		select {
		case <-wd.ctx.Done():
			err = ErrCanceled
			_ = wd.Quit(context.Background())
			return
		default:
		}
	}()

	if Log != nil {
		Log.Printf("-> %s %s [%d bytes]", method, url, len(data))
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", jsonMIMEType)
	if method == "POST" {
		req.Header.Add("Content-Type", jsonMIMEType)
	}

	if Trace {
		if dump, err := httputil.DumpRequest(req, true); err == nil && Log != nil {
			Log.Printf("-> TRACE\n%s", dump)
		}
	}

	req = req.WithContext(wd.ctx)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if Trace {
		if dump, err := httputil.DumpResponse(res, true); err == nil && Log != nil {
			Log.Printf("<- TRACE\n%s", dump)
		}
	}

	buf, err = ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if Log != nil {
		Log.Printf("<- %s (%s) [%d bytes]", res.Status, res.Header["Content-Type"], len(buf))
	}

	pE := func(r *reply) error {
		sr := &replyValue{}
		var backendError string
		err = json.Unmarshal([]byte(r.Value), sr)
		if err == nil {
			// can analyze the error
			if sr.Message != "" {
				rm := &replyMessage{}
				err = json.Unmarshal([]byte(sr.Message), rm)
				if err == nil {
					backendError = rm.ErrorMessage
				}
			}
		}

		message, ok := errorCodes[r.Status]
		if !ok {
			message = fmt.Sprintf("unknown error - %d", r.Status)
		}

		return fmt.Errorf("%v%v", message, " - "+fmt.Sprintf("%q", backendError))
	}

	if res.StatusCode >= 400 {
		reply := new(reply)
		err := json.Unmarshal(buf, reply)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Bad server reply status: %s", res.Status))
		}
		errParsed := pE(reply)

		return nil, errParsed
	}

	/* Some bug(?) in Selenium gets us nil values in output, json.Unmarshal is
	* not happy about that.
	 */
	if strings.HasPrefix(res.Header.Get("Content-Type"), jsonMIMEType) {
		reply := new(reply)
		err := json.Unmarshal(buf, reply)
		if err != nil {
			return nil, err
		}

		if reply.Status != SUCCESS {

			errParsed := pE(reply)
			return nil, errParsed
		}
		return buf, err
	}

	// Nothing was returned, this is OK for some commands
	return buf, nil
}

var httpClient = http.Client{
	// WebDriver requires that all requests have an 'Accept: application/json' header. We must add
	// it here because by default net/http will not include that header when following redirects.
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		req.Header.Add("Accept", jsonMIMEType)
		if Trace {
			if dump, err := httputil.DumpRequest(req, true); err == nil && Log != nil {
				Log.Printf("-> TRACE (redirected request)\n%s", dump)
			}
		}
		return nil
	},
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 30 * time.Second,
	},
	Timeout: 60 * time.Second,
}

// Server reply to WebDriver command.
type reply struct {
	SessionId string
	Status    int
	Value     json.RawMessage
}

type replyValue struct {
	Message string `json:"message"`
}

type replyMessage struct {
	ErrorMessage string `json:"errorMessage"`
}

func (r *reply) readValue(v interface{}) error {
	return json.Unmarshal(r.Value, v)
}

// An active session.
type Session struct {
	Id           string
	Capabilities Capabilities
}

/* Create new remote client, this will also start a new session.
   capabilities - the desired capabilities, see http://goo.gl/SNlAk
   executor - the URL to the Selenim server
*/
func NewRemote(ctx context.Context, capabilities Capabilities, executor string) (WebDriver, error) {
	if executor == "" {
		executor = defaultExecutor
	}

	wd := &remoteWebDriver{
		executor:     executor,
		capabilities: capabilities,
		ctx:          context.Background(),
	}
	// FIXME: Handle profile

	_, err := wd.NewSession(ctx)
	if err != nil {
		return nil, err
	}

	return wd, nil
}

func (wd *remoteWebDriver) stringCommand(ctx context.Context, urlTemplate string) (v string, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url(urlTemplate, wd.id), nil); err == nil {
		err = r.readValue(&v)
	}
	return
}

func (wd *remoteWebDriver) voidCommand(ctx context.Context, urlTemplate string, params interface{}) (err error) {
	var data []byte
	if params != nil {
		data, err = json.Marshal(params)
	}
	if err == nil {
		_, err = wd.send(ctx, "POST", wd.url(urlTemplate, wd.id), data)
	}
	return

}

func (wd remoteWebDriver) stringsCommand(ctx context.Context, urlTemplate string) (v []string, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url(urlTemplate, wd.id), nil); err == nil {
		err = r.readValue(&v)
	}
	return
}

func (wd *remoteWebDriver) boolCommand(ctx context.Context, urlTemplate string) (v bool, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url(urlTemplate, wd.id), nil); err == nil {
		err = r.readValue(&v)
	}
	return
}

// WebDriver interface implementation

func (wd *remoteWebDriver) Status(ctx context.Context) (v *Status, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url("/status"), nil); err == nil {
		err = r.readValue(&v)
	}
	return
}

func (wd *remoteWebDriver) Sessions(ctx context.Context) (sessions []Session, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url("/sessions"), nil); err == nil {
		err = r.readValue(&sessions)
	}
	return
}

func (wd *remoteWebDriver) NewSession(ctx context.Context) (string, error) {
	message := map[string]interface{}{
		"desiredCapabilities": wd.capabilities,
	}

	var data []byte
	data, err := json.Marshal(message)
	if err != nil {
		return "", err
	}

	r, err := wd.send(ctx, "POST", wd.url("/session"), data)
	if err != nil {
		return "", err
	}
	wd.id = r.SessionId

	return r.SessionId, nil
}

func (wd *remoteWebDriver) Capabilities(ctx context.Context) (v Capabilities, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url("/session/%s", wd.id), nil); err == nil {
		r.readValue(&v)
	}
	return
}

func (wd *remoteWebDriver) GetSessionID() string {
	return wd.id
}

func (wd *remoteWebDriver) SetTimeout(ctx context.Context, timeoutType string, ms uint) error {
	params := map[string]interface{}{"type": timeoutType, "ms": ms}
	return wd.voidCommand(ctx, "/session/%s/timeouts", params)
}

func (wd *remoteWebDriver) SetAsyncScriptTimeout(ctx context.Context, ms uint) error {
	params := map[string]uint{"ms": ms}
	return wd.voidCommand(ctx, "/session/%s/timeouts/async_script", params)
}

func (wd *remoteWebDriver) SetImplicitWaitTimeout(ctx context.Context, ms uint) error {
	params := map[string]uint{"ms": ms}
	return wd.voidCommand(ctx, "/session/%s/timeouts/implicit_wait", params)
}

func (wd *remoteWebDriver) AvailableEngines(ctx context.Context) ([]string, error) {
	return wd.stringsCommand(ctx, "/session/%s/ime/available_engines")
}

func (wd *remoteWebDriver) ActiveEngine(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/ime/active_engine")
}

func (wd *remoteWebDriver) IsEngineActivated(ctx context.Context) (bool, error) {
	return wd.boolCommand(ctx, "/session/%s/ime/activated")
}

func (wd *remoteWebDriver) DeactivateEngine(ctx context.Context) error {
	return wd.voidCommand(ctx, "session/%s/ime/deactivate", nil)
}

func (wd *remoteWebDriver) ActivateEngine(ctx context.Context, engine string) (err error) {
	return wd.voidCommand(ctx, "/session/%s/ime/activate", map[string]string{"engine": engine})
}

func (wd *remoteWebDriver) Quit(ctx context.Context) (err error) {
	wd.haveQuitMu.Lock()
	defer wd.haveQuitMu.Unlock()
	if wd.haveQuit {
		// Double-Quit is an error-free no-op.
		return nil
	}
	wd.haveQuit = true
	// Quit is the one method which cannot be canceled.
	// It's also the last thing that happens in a webdriver, so we can
	// kill the context here.
	wd.ctx = context.Background()

	if _, err = wd.execute(ctx, "DELETE", wd.url("/session/%s", wd.id), nil); err == nil {
		wd.id = ""
	}
	return
}

func (wd *remoteWebDriver) CurrentWindowHandle(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/window_handle")
}

func (wd *remoteWebDriver) WindowHandles(ctx context.Context) ([]string, error) {
	return wd.stringsCommand(ctx, "/session/%s/window_handles")
}

func (wd *remoteWebDriver) CurrentURL(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/url")
}

func (wd *remoteWebDriver) Get(ctx context.Context, url string) error {
	return wd.voidCommand(ctx, "/session/%s/url", map[string]string{"url": url})
}

func (wd *remoteWebDriver) Forward(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/forward", nil)
}

func (wd *remoteWebDriver) Back(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/back", nil)
}

func (wd *remoteWebDriver) Refresh(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/refresh", nil)
}

func (wd *remoteWebDriver) Title(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/title")
}

func (wd *remoteWebDriver) PageSource(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/source")
}

type element struct {
	Element string `json:"ELEMENT"`
}

func (wd *remoteWebDriver) find(ctx context.Context, by, value, suffix, url string) (r *reply, err error) {
	params := map[string]string{"using": by, "value": value}
	var data []byte
	if data, err = json.Marshal(params); err == nil {
		if url == "" {
			url = "/session/%s/element"
		}
		urlTemplate := url + suffix
		url = wd.url(urlTemplate, wd.id)
		r, err = wd.send(ctx, "POST", url, data)
	}
	return
}

func decodeElement(wd *remoteWebDriver, r *reply) WebElement {
	var elem element
	if err := r.readValue(&elem); err != nil {
		panic(err.Error() + ": " + string(r.Value))
	}
	return &remoteWE{parent: wd, id: elem.Element}
}

func (wd *remoteWebDriver) FindElement(ctx context.Context, by, value string) (WebElement, error) {
	if res, err := wd.find(ctx, by, value, "", ""); err == nil {
		return decodeElement(wd, res), nil
	} else {
		return nil, err
	}
}

func decodeElements(wd *remoteWebDriver, r *reply) (welems []WebElement) {
	var elems []element
	if err := r.readValue(&elems); err != nil {
		panic(err.Error() + ": " + string(r.Value))
	}
	for _, elem := range elems {
		welems = append(welems, &remoteWE{wd, elem.Element})
	}
	return
}

func (wd *remoteWebDriver) FindElements(ctx context.Context, by, value string) ([]WebElement, error) {
	if res, err := wd.find(ctx, by, value, "s", ""); err == nil {
		return decodeElements(wd, res), nil
	} else {
		return nil, err
	}
}

func (wd *remoteWebDriver) Q(ctx context.Context, sel string) (WebElement, error) {
	return wd.FindElement(ctx, ByCSSSelector, sel)
}

func (wd *remoteWebDriver) QAll(ctx context.Context, sel string) ([]WebElement, error) {
	return wd.FindElements(ctx, ByCSSSelector, sel)
}

func (wd *remoteWebDriver) Close(ctx context.Context) error {
	_, err := wd.execute(ctx, "DELETE", wd.url("/session/%s/window", wd.id), nil)
	return err
}

func (wd *remoteWebDriver) SwitchWindow(ctx context.Context, name string) error {
	if name == "" {
		name = "current"
	}
	params := map[string]string{"name": name}
	return wd.voidCommand(ctx, "/session/%s/window", params)
}

func (wd *remoteWebDriver) CloseWindow(ctx context.Context, name string) error {
	_, err := wd.execute(ctx, "DELETE", wd.url("/session/%s/window", wd.id), nil)
	return err
}

func (wd *remoteWebDriver) WindowSize(ctx context.Context, name string) (sz *Size, err error) {
	if name == "" {
		name = "current"
	}
	url := wd.url("/session/%s/window/%s/size", wd.id, name)
	var r *reply
	if r, err = wd.send(ctx, "GET", url, nil); err == nil {
		err = r.readValue(&sz)
	}
	return
}

func (wd *remoteWebDriver) WindowPosition(ctx context.Context, name string) (pt *Point, err error) {
	if name == "" {
		name = "current"
	}
	url := wd.url("/session/%s/window/%s/position", wd.id, name)
	var r *reply
	if r, err = wd.send(ctx, "GET", url, nil); err == nil {
		err = r.readValue(&pt)
	}
	return
}

func (wd *remoteWebDriver) ResizeWindow(ctx context.Context, name string, to Size) error {
	if name == "" {
		name = "current"
	}
	url := wd.url("/session/%s/window/%s/size", wd.id, name)
	data, err := json.Marshal(to)
	if err != nil {
		return err
	}
	_, err = wd.send(ctx, "POST", url, data)
	return err
}

func (wd *remoteWebDriver) SwitchFrame(ctx context.Context, frame string) error {
	params := map[string]string{"id": frame}
	return wd.voidCommand(ctx, "/session/%s/frame", params)
}

func (wd *remoteWebDriver) SwitchFrameParent(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/frame/parent", nil)
}

func (wd *remoteWebDriver) ActiveElement(ctx context.Context) (WebElement, error) {
	url := wd.url("/session/%s/element/active", wd.id)
	if r, err := wd.send(ctx, "GET", url, nil); err == nil {
		return decodeElement(wd, r), nil
	} else {
		return nil, err
	}
}

func (wd *remoteWebDriver) GetCookies(ctx context.Context) (c []Cookie, err error) {
	var r *reply
	if r, err = wd.send(ctx, "GET", wd.url("/session/%s/cookie", wd.id), nil); err == nil {
		err = r.readValue(&c)
		if err == nil {
			parseCookieExpiry(&c, r.Value)
		}
	}
	return
}

func parseCookieExpiry(cookies *[]Cookie, raw json.RawMessage) {
	var expiries []struct {
		Expiry json.Number
	}

	err := json.Unmarshal(raw, &expiries)
	if err != nil {
		return
	}

	for i, _ := range *cookies {
		expiry, err := expiries[i].Expiry.Float64()
		if err != nil {
			continue
		}

		(*cookies)[i].Expiry = uint(expiry)
	}
}

func (wd *remoteWebDriver) AddCookie(ctx context.Context, cookie *Cookie) error {
	params := map[string]*Cookie{"cookie": cookie}
	return wd.voidCommand(ctx, "/session/%s/cookie", params)
}

func (wd *remoteWebDriver) DeleteAllCookies(ctx context.Context) error {
	_, err := wd.execute(ctx, "DELETE", wd.url("/session/%s/cookie", wd.id), nil)
	return err
}

func (wd *remoteWebDriver) DeleteCookie(ctx context.Context, name string) error {
	_, err := wd.execute(ctx, "DELETE", wd.url("/session/%s/cookie/%s", wd.id, name), nil)
	return err
}

func (wd *remoteWebDriver) Click(ctx context.Context, button int) error {
	params := map[string]int{"button": button}
	return wd.voidCommand(ctx, "/session/%s/click", params)
}

func (wd *remoteWebDriver) DoubleClick(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/doubleclick", nil)
}

func (wd *remoteWebDriver) ButtonDown(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/buttondown", nil)
}

func (wd *remoteWebDriver) ButtonUp(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/buttonup", nil)
}

func (wd *remoteWebDriver) SendModifier(ctx context.Context, modifier string, isDown bool) error {
	params := map[string]interface{}{
		"value":  modifier,
		"isdown": isDown,
	}

	data, err := json.Marshal(params)
	if err != nil {
		return err
	}

	return wd.voidCommand(ctx, "/session/%s/modifier", data)
}

func (wd *remoteWebDriver) DismissAlert(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/dismiss_alert", nil)
}

func (wd *remoteWebDriver) AcceptAlert(ctx context.Context) error {
	return wd.voidCommand(ctx, "/session/%s/accept_alert", nil)
}

func (wd *remoteWebDriver) AlertText(ctx context.Context) (string, error) {
	return wd.stringCommand(ctx, "/session/%s/alert_text")
}

func (wd *remoteWebDriver) SetAlertText(ctx context.Context, text string) error {
	params := map[string]string{"text": text}
	return wd.voidCommand(ctx, "/session/%s/alert_text", params)
}

func (wd *remoteWebDriver) execScript(ctx context.Context, script string, args []interface{}, suffix string) (res interface{}, err error) {
	if args == nil {
		args = []interface{}{}
	}
	for i, arg := range args {
		if v, ok := arg.(*remoteWE); ok {
			args[i] = &element{Element: v.id}
		}
	}
	params := map[string]interface{}{
		"script": script,
		"args":   args,
	}
	var data []byte
	if data, err = json.Marshal(params); err != nil {
		return nil, err
	}
	url := wd.url("/session/%s/execute"+suffix, wd.id)
	var r *reply
	if r, err = wd.send(ctx, "POST", url, data); err == nil {
		err = r.readValue(&res)
		if err != nil {
			return
		}

	}
	return
}

func (wd *remoteWebDriver) ExecuteScript(ctx context.Context, script string, args []interface{}) (interface{}, error) {
	return wd.execScript(ctx, script, args, "")
}

func (wd *remoteWebDriver) ExecuteScriptAsync(ctx context.Context, script string, args []interface{}) (interface{}, error) {
	return wd.execScript(ctx, script, args, "_async")
}

func (wd *remoteWebDriver) Screenshot(ctx context.Context) (io.Reader, error) {
	data, err := wd.stringCommand(ctx, "/session/%s/screenshot")
	if err != nil {
		return nil, err
	}

	// Selenium returns base64 encoded image
	buf := []byte(data)
	decoder := base64.NewDecoder(base64.StdEncoding, bytes.NewBuffer(buf))
	return decoder, nil
}

func (wd *remoteWebDriver) T(t TestingT) WebDriverT {
	return &webDriverT{wd, t}
}

// WebElement interface implementation

type remoteWE struct {
	parent *remoteWebDriver
	id     string
}

func (elem *remoteWE) Click(ctx context.Context) error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/click", elem.id)
	return elem.parent.voidCommand(ctx, urlTemplate, nil)
}

func (elem *remoteWE) SendKeys(ctx context.Context, keys string) error {
	chars := make([]string, len(keys))
	for i, c := range keys {
		chars[i] = string(c)
	}
	params := map[string][]string{"value": chars}
	urltmpl := fmt.Sprintf("/session/%%s/element/%s/value", elem.id)
	return elem.parent.voidCommand(ctx, urltmpl, params)
}

func (elem *remoteWE) TagName(ctx context.Context) (string, error) {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/name", elem.id)
	return elem.parent.stringCommand(ctx, urlTemplate)
}

func (elem *remoteWE) Text(ctx context.Context) (string, error) {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/text", elem.id)
	return elem.parent.stringCommand(ctx, urlTemplate)
}

func (elem *remoteWE) Submit(ctx context.Context) error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/submit", elem.id)
	return elem.parent.voidCommand(ctx, urlTemplate, nil)
}

func (elem *remoteWE) Clear(ctx context.Context) error {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/clear", elem.id)
	return elem.parent.voidCommand(ctx, urlTemplate, nil)
}

func (elem *remoteWE) MoveTo(ctx context.Context, xOffset, yOffset int) error {
	params := map[string]interface{}{
		"element": elem.id,
		"xoffset": xOffset,
		"yoffset": yOffset,
	}
	return elem.parent.voidCommand(ctx, "/session/%s/moveto", params)
}

func (elem *remoteWE) FindElement(ctx context.Context, by, value string) (WebElement, error) {
	res, err := elem.parent.find(ctx, by, value, "", fmt.Sprintf("/session/%%s/element/%s/element", elem.id))
	if err != nil {
		return nil, err
	}
	return decodeElement(elem.parent, res), nil
}

func (elem *remoteWE) Q(ctx context.Context, sel string) (WebElement, error) {
	return elem.FindElement(ctx, ByCSSSelector, sel)
}

func (elem *remoteWE) QAll(ctx context.Context, sel string) ([]WebElement, error) {
	return elem.FindElements(ctx, ByCSSSelector, sel)
}

func (elem *remoteWE) FindElements(ctx context.Context, by, value string) ([]WebElement, error) {
	res, err := elem.parent.find(ctx, by, value, "s", fmt.Sprintf("/session/%%s/element/%s/element", elem.id))
	if err != nil {
		return nil, err
	}
	return decodeElements(elem.parent, res), nil
}

func (elem *remoteWE) boolQuery(ctx context.Context, urlTemplate string) (bool, error) {
	url := fmt.Sprintf(urlTemplate, elem.id)
	return elem.parent.boolCommand(ctx, url)
}

// Porperties
func (elem *remoteWE) IsSelected(ctx context.Context) (bool, error) {
	return elem.boolQuery(ctx, "/session/%%s/element/%s/selected")
}

func (elem *remoteWE) IsEnabled(ctx context.Context) (bool, error) {
	return elem.boolQuery(ctx, "/session/%%s/element/%s/enabled")
}

func (elem *remoteWE) IsDisplayed(ctx context.Context) (bool, error) {
	return elem.boolQuery(ctx, "/session/%%s/element/%s/displayed")
}

func (elem *remoteWE) GetAttribute(ctx context.Context, name string) (string, error) {
	template := "/session/%%s/element/%s/attribute/%s"
	urlTemplate := fmt.Sprintf(template, elem.id, name)

	return elem.parent.stringCommand(ctx, urlTemplate)
}

func (elem *remoteWE) location(ctx context.Context, suffix string) (pt *Point, err error) {
	wd := elem.parent
	path := "/session/%s/element/%s/location" + suffix
	url := wd.url(path, wd.id, elem.id)
	var r *reply
	if r, err = wd.send(ctx, "GET", url, nil); err == nil {
		err = r.readValue(&pt)
	}
	return
}

func (elem *remoteWE) Location(ctx context.Context) (*Point, error) {
	return elem.location(ctx, "")
}

func (elem *remoteWE) LocationInView(ctx context.Context) (*Point, error) {
	return elem.location(ctx, "_in_view")
}

func (elem *remoteWE) Size(ctx context.Context) (sz *Size, err error) {
	wd := elem.parent
	url := wd.url("/session/%s/element/%s/size", wd.id, elem.id)
	var r *reply
	if r, err = wd.send(ctx, "GET", url, nil); err == nil {
		err = r.readValue(&sz)
	}
	return
}

func (elem *remoteWE) CSSProperty(ctx context.Context, name string) (string, error) {
	urlTemplate := fmt.Sprintf("/session/%%s/element/%s/css/%s", elem.id, name)
	return elem.parent.stringCommand(ctx, urlTemplate)
}

func (elem *remoteWE) T(t TestingT) WebElementT {
	return &webElementT{elem, t}
}
