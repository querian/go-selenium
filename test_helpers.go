package selenium

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// A single-return-value interface to WebDriverT that is useful when using WebDrivers in test code.
// Obtain a WebDriverT by calling webDriver.T(t), where t *testing.T is the test handle for the
// current test. The methods of WebDriverT call wt.t.Fatalf upon encountering errors instead of using
// multiple returns to indicate errors.
type WebDriverT interface {
	WebDriver() WebDriver

	NewSession(ctx context.Context) string

	SetTimeout(ctx context.Context, timeoutType string, ms uint)
	SetAsyncScriptTimeout(ctx context.Context, ms uint)
	SetImplicitWaitTimeout(ctx context.Context, ms uint)

	Quit(ctx context.Context)

	CurrentWindowHandle(ctx context.Context) string
	WindowHandles(ctx context.Context) []string
	CurrentURL(ctx context.Context) string
	Title(ctx context.Context) string
	PageSource(ctx context.Context) string
	Close(ctx context.Context)
	SwitchFrame(ctx context.Context, frame string)
	SwitchFrameParent(ctx context.Context)
	SwitchWindow(ctx context.Context, name string)
	CloseWindow(ctx context.Context, name string)
	WindowSize(ctx context.Context, name string) *Size
	WindowPosition(ctx context.Context, name string) *Point
	ResizeWindow(ctx context.Context, name string, to Size)

	Get(ctx context.Context, url string)
	Forward(ctx context.Context)
	Back(ctx context.Context)
	Refresh(ctx context.Context)

	FindElement(ctx context.Context, by, value string) WebElementT
	FindElements(ctx context.Context, by, value string) []WebElementT
	ActiveElement(ctx context.Context) WebElement

	// Shortcut for FindElement(ByCSSSelector, sel)
	Q(ctx context.Context, sel string) WebElementT
	// Shortcut for FindElements(ByCSSSelector, sel)
	QAll(ctx context.Context, sel string) []WebElementT

	GetCookies(ctx context.Context) []Cookie
	AddCookie(ctx context.Context, cookie *Cookie)
	DeleteAllCookies(ctx context.Context)
	DeleteCookie(ctx context.Context, name string)

	Click(ctx context.Context, button int)
	DoubleClick(ctx context.Context)
	ButtonDown(ctx context.Context)
	ButtonUp(ctx context.Context)

	SendModifier(ctx context.Context, modifier string, isDown bool)
	Screenshot(ctx context.Context) io.Reader

	DismissAlert(ctx context.Context)
	AcceptAlert(ctx context.Context)
	AlertText(ctx context.Context) string
	SetAlertText(ctx context.Context, text string)

	ExecuteScript(ctx context.Context, script string, args []interface{}) interface{}
	ExecuteScriptAsync(ctx context.Context, script string, args []interface{}) interface{}
}

type webDriverT struct {
	d WebDriver
	t TestingT
}

func (wt *webDriverT) WebDriver() WebDriver {
	return wt.d
}

func (wt *webDriverT) NewSession(ctx context.Context) (id string) {
	var err error
	if id, err = wt.d.NewSession(ctx); err != nil {
		fatalf(wt.t, "NewSession: %s", err)
	}
	return
}

func (wt *webDriverT) SetTimeout(ctx context.Context, timeoutType string, ms uint) {
	if err := wt.d.SetTimeout(ctx, timeoutType, ms); err != nil {
		fatalf(wt.t, "SetTimeout(timeoutType=%q, ms=%d): %s", timeoutType, ms, err)
	}
}

func (wt *webDriverT) SetAsyncScriptTimeout(ctx context.Context, ms uint) {
	if err := wt.d.SetAsyncScriptTimeout(ctx, ms); err != nil {
		fatalf(wt.t, "SetAsyncScriptTimeout(%d msec): %s", ms, err)
	}
}

func (wt *webDriverT) SetImplicitWaitTimeout(ctx context.Context, ms uint) {
	if err := wt.d.SetImplicitWaitTimeout(ctx, ms); err != nil {
		fatalf(wt.t, "SetImplicitWaitTimeout(%d msec): %s", ms, err)
	}
}

func (wt *webDriverT) Quit(ctx context.Context) {
	if err := wt.d.Quit(ctx); err != nil {
		fatalf(wt.t, "Quit: %s", err)
	}
}

func (wt *webDriverT) CurrentWindowHandle(ctx context.Context) (v string) {
	var err error
	if v, err = wt.d.CurrentWindowHandle(ctx); err != nil {
		fatalf(wt.t, "CurrentWindowHandle: %s", err)
	}
	return
}

func (wt *webDriverT) WindowHandles(ctx context.Context) (hs []string) {
	var err error
	if hs, err = wt.d.WindowHandles(ctx); err != nil {
		fatalf(wt.t, "WindowHandles: %s", err)
	}
	return
}

func (wt *webDriverT) CurrentURL(ctx context.Context) (v string) {
	var err error
	if v, err = wt.d.CurrentURL(ctx); err != nil {
		fatalf(wt.t, "CurrentURL: %s", err)
	}
	return
}

func (wt *webDriverT) Title(ctx context.Context) (v string) {
	var err error
	if v, err = wt.d.Title(ctx); err != nil {
		fatalf(wt.t, "Title: %s", err)
	}
	return
}

func (wt *webDriverT) PageSource(ctx context.Context) (v string) {
	var err error
	if v, err = wt.d.PageSource(ctx); err != nil {
		fatalf(wt.t, "PageSource: %s", err)
	}
	return
}

func (wt *webDriverT) Close(ctx context.Context) {
	if err := wt.d.Close(ctx); err != nil {
		fatalf(wt.t, "Close: %s", err)
	}
}

func (wt *webDriverT) SwitchFrame(ctx context.Context, frame string) {
	if err := wt.d.SwitchFrame(ctx, frame); err != nil {
		fatalf(wt.t, "SwitchFrame(%q): %s", frame, err)
	}
}

func (wt *webDriverT) SwitchFrameParent(ctx context.Context) {
	if err := wt.d.SwitchFrameParent(ctx); err != nil {
		fatalf(wt.t, "SwitchFrameParent(): %s", err)
	}
}

func (wt *webDriverT) SwitchWindow(ctx context.Context, name string) {
	if err := wt.d.SwitchWindow(ctx, name); err != nil {
		fatalf(wt.t, "SwitchWindow(%q): %s", name, err)
	}
}

func (wt *webDriverT) CloseWindow(ctx context.Context, name string) {
	if err := wt.d.CloseWindow(ctx, name); err != nil {
		fatalf(wt.t, "CloseWindow(%q): %s", name, err)
	}
}

func (wt *webDriverT) WindowSize(ctx context.Context, name string) *Size {
	sz, err := wt.d.WindowSize(ctx, name)
	if err != nil {
		fatalf(wt.t, "WindowSize(%q): %s", name, err)
	}
	return sz
}

func (wt *webDriverT) WindowPosition(ctx context.Context, name string) *Point {
	pt, err := wt.d.WindowPosition(ctx, name)
	if err != nil {
		fatalf(wt.t, "WindowPosition(%q): %s", name, err)
	}
	return pt
}

func (wt *webDriverT) ResizeWindow(ctx context.Context, name string, to Size) {
	if err := wt.d.ResizeWindow(ctx, name, to); err != nil {
		fatalf(wt.t, "ResizeWindow(%s, %+v): %s", name, to, err)
	}
}

func (wt *webDriverT) Get(ctx context.Context, name string) {
	if err := wt.d.Get(ctx, name); err != nil {
		fatalf(wt.t, "Get(%q): %s", name, err)
	}
}

func (wt *webDriverT) Forward(ctx context.Context) {
	if err := wt.d.Forward(ctx); err != nil {
		fatalf(wt.t, "Forward: %s", err)
	}
}

func (wt *webDriverT) Back(ctx context.Context) {
	if err := wt.d.Back(ctx); err != nil {
		fatalf(wt.t, "Back: %s", err)
	}
}

func (wt *webDriverT) Refresh(ctx context.Context) {
	if err := wt.d.Refresh(ctx); err != nil {
		fatalf(wt.t, "Refresh: %s", err)
	}
}

func (wt *webDriverT) FindElement(ctx context.Context, by, value string) (elem WebElementT) {
	if elem_, err := wt.d.FindElement(ctx, by, value); err == nil {
		elem = elem_.T(wt.t)
	} else {
		fatalf(wt.t, "FindElement(by=%q, value=%q): %s", by, value, err)
	}
	return
}

func (wt *webDriverT) FindElements(ctx context.Context, by, value string) (elems []WebElementT) {
	if elems_, err := wt.d.FindElements(ctx, by, value); err == nil {
		for _, elem := range elems_ {
			elems = append(elems, elem.T(wt.t))
		}
	} else {
		fatalf(wt.t, "FindElements(by=%q, value=%q): %s", by, value, err)
	}
	return
}

func (wt *webDriverT) Q(ctx context.Context, sel string) (elem WebElementT) {
	return wt.FindElement(ctx, ByCSSSelector, sel)
}

func (wt *webDriverT) QAll(ctx context.Context, sel string) (elems []WebElementT) {
	return wt.FindElements(ctx, ByCSSSelector, sel)
}

func (wt *webDriverT) ActiveElement(ctx context.Context) (elem WebElement) {
	var err error
	if elem, err = wt.d.ActiveElement(ctx); err != nil {
		fatalf(wt.t, "ActiveElement: %s", err)
	}
	return
}

func (wt *webDriverT) GetCookies(ctx context.Context) (c []Cookie) {
	var err error
	if c, err = wt.d.GetCookies(ctx); err != nil {
		fatalf(wt.t, "GetCookies: %s", err)
	}
	return
}

func (wt *webDriverT) AddCookie(ctx context.Context, cookie *Cookie) {
	if err := wt.d.AddCookie(ctx, cookie); err != nil {
		fatalf(wt.t, "AddCookie(%+q): %s", cookie, err)
	}
	return
}

func (wt *webDriverT) DeleteAllCookies(ctx context.Context) {
	if err := wt.d.DeleteAllCookies(ctx); err != nil {
		fatalf(wt.t, "DeleteAllCookies: %s", err)
	}
}

func (wt *webDriverT) DeleteCookie(ctx context.Context, name string) {
	if err := wt.d.DeleteCookie(ctx, name); err != nil {
		fatalf(wt.t, "DeleteCookie(%q): %s", name, err)
	}
}

func (wt *webDriverT) Click(ctx context.Context, button int) {
	if err := wt.d.Click(ctx, button); err != nil {
		fatalf(wt.t, "Click(%d): %s", button, err)
	}
}

func (wt *webDriverT) DoubleClick(ctx context.Context) {
	if err := wt.d.DoubleClick(ctx); err != nil {
		fatalf(wt.t, "DoubleClick: %s", err)
	}
}

func (wt *webDriverT) ButtonDown(ctx context.Context) {
	if err := wt.d.ButtonDown(ctx); err != nil {
		fatalf(wt.t, "ButtonDown: %s", err)
	}
}

func (wt *webDriverT) ButtonUp(ctx context.Context) {
	if err := wt.d.ButtonUp(ctx); err != nil {
		fatalf(wt.t, "ButtonUp: %s", err)
	}
}

func (wt *webDriverT) SendModifier(ctx context.Context, modifier string, isDown bool) {
	if err := wt.d.SendModifier(ctx, modifier, isDown); err != nil {
		fatalf(wt.t, "SendModifier(modifier=%q, isDown=%s): %s", modifier, isDown, err)
	}
}

func (wt *webDriverT) Screenshot(ctx context.Context) (data io.Reader) {
	var err error
	if data, err = wt.d.Screenshot(ctx); err != nil {
		fatalf(wt.t, "Screenshot: %s", err)
	}
	return
}

func (wt *webDriverT) DismissAlert(ctx context.Context) {
	if err := wt.d.DismissAlert(ctx); err != nil {
		fatalf(wt.t, "DismissAlert: %s", err)
	}
}

func (wt *webDriverT) AcceptAlert(ctx context.Context) {
	if err := wt.d.AcceptAlert(ctx); err != nil {
		fatalf(wt.t, "AcceptAlert: %s", err)
	}
}

func (wt *webDriverT) AlertText(ctx context.Context) (text string) {
	var err error
	if text, err = wt.d.AlertText(ctx); err != nil {
		fatalf(wt.t, "AlertText: %s", err)
	}
	return
}

func (wt *webDriverT) SetAlertText(ctx context.Context, text string) {
	var err error
	if err = wt.d.SetAlertText(ctx, text); err != nil {
		fatalf(wt.t, "SetAlertText(%q): %s", text, err)
	}
}

func (wt *webDriverT) ExecuteScript(ctx context.Context, script string, args []interface{}) (res interface{}) {
	var err error
	if res, err = wt.d.ExecuteScript(ctx, script, args); err != nil {
		fatalf(wt.t, "ExecuteScript(script=%q, args=%+q): %s", script, args, err)
	}
	return
}

func (wt *webDriverT) ExecuteScriptAsync(ctx context.Context, script string, args []interface{}) (res interface{}) {
	var err error
	if res, err = wt.d.ExecuteScriptAsync(ctx, script, args); err != nil {
		fatalf(wt.t, "ExecuteScriptAsync(script=%q, args=%+q): %s", script, args, err)
	}
	return
}

// A single-return-value interface to WebElement that is useful when using WebElements in test code.
// Obtain a WebElementT by calling webElement.T(t), where t *testing.T is the test handle for the
// current test. The methods of WebElementT call wt.fatalf upon encountering errors instead of using
// multiple returns to indicate errors.
type WebElementT interface {
	WebElement() WebElement

	Click(ctx context.Context)
	SendKeys(ctx context.Context, keys string)
	Submit(ctx context.Context)
	Clear(ctx context.Context)
	MoveTo(ctx context.Context, xOffset, yOffset int)

	FindElement(ctx context.Context, by, value string) WebElementT
	FindElements(ctx context.Context, by, value string) []WebElementT

	// Shortcut for FindElement(ByCSSSelector, sel)
	Q(ctx context.Context, sel string) WebElementT
	// Shortcut for FindElements(ByCSSSelector, sel)
	QAll(ctx context.Context, sel string) []WebElementT

	TagName(ctx context.Context) string
	Text(ctx context.Context) string
	IsSelected(ctx context.Context) bool
	IsEnabled(ctx context.Context) bool
	IsDisplayed(ctx context.Context) bool
	GetAttribute(ctx context.Context, name string) string
	Location(ctx context.Context) *Point
	LocationInView(ctx context.Context) *Point
	Size(ctx context.Context) *Size
	CSSProperty(ctx context.Context, name string) string
}

type webElementT struct {
	e WebElement
	t TestingT
}

func (wt *webElementT) WebElement() WebElement {
	return wt.e
}

func (wt *webElementT) Click(ctx context.Context) {
	if err := wt.e.Click(ctx); err != nil {
		fatalf(wt.t, "Click: %s", err)
	}
}

func (wt *webElementT) SendKeys(ctx context.Context, keys string) {
	if err := wt.e.SendKeys(ctx, keys); err != nil {
		fatalf(wt.t, "SendKeys(%q): %s", keys, err)
	}
}

func (wt *webElementT) Submit(ctx context.Context) {
	if err := wt.e.Submit(ctx); err != nil {
		fatalf(wt.t, "Submit: %s", err)
	}
}

func (wt *webElementT) Clear(ctx context.Context) {
	if err := wt.e.Clear(ctx); err != nil {
		fatalf(wt.t, "Clear: %s", err)
	}
}

func (wt *webElementT) MoveTo(ctx context.Context, xOffset, yOffset int) {
	if err := wt.e.MoveTo(ctx, xOffset, yOffset); err != nil {
		fatalf(wt.t, "MoveTo(xOffset=%d, yOffset=%d): %s", xOffset, yOffset, err)
	}
}

func (wt *webElementT) FindElement(ctx context.Context, by, value string) WebElementT {
	if elem, err := wt.e.FindElement(ctx, by, value); err == nil {
		return elem.T(wt.t)
	} else {
		fatalf(wt.t, "FindElement(by=%q, value=%q): %s", by, value, err)
		panic("unreachable")
	}
}

func (wt *webElementT) FindElements(ctx context.Context, by, value string) []WebElementT {
	if elems, err := wt.e.FindElements(ctx, by, value); err == nil {
		elemsT := make([]WebElementT, len(elems))
		for i, elem := range elems {
			elemsT[i] = elem.T(wt.t)
		}
		return elemsT
	} else {
		fatalf(wt.t, "FindElements(by=%q, value=%q): %s", by, value, err)
		panic("unreachable")
	}
}

func (wt *webElementT) Q(ctx context.Context, sel string) (elem WebElementT) {
	return wt.FindElement(ctx, ByCSSSelector, sel)
}

func (wt *webElementT) QAll(ctx context.Context, sel string) (elems []WebElementT) {
	return wt.FindElements(ctx, ByCSSSelector, sel)
}

func (wt *webElementT) TagName(ctx context.Context) (v string) {
	var err error
	if v, err = wt.e.TagName(ctx); err != nil {
		fatalf(wt.t, "TagName: %s", err)
	}
	return
}

func (wt *webElementT) Text(ctx context.Context) (v string) {
	var err error
	if v, err = wt.e.Text(ctx); err != nil {
		fatalf(wt.t, "Text: %s", err)
	}
	return
}

func (wt *webElementT) IsSelected(ctx context.Context) (v bool) {
	var err error
	if v, err = wt.e.IsSelected(ctx); err != nil {
		fatalf(wt.t, "IsSelected: %s", err)
	}
	return
}

func (wt *webElementT) IsEnabled(ctx context.Context) (v bool) {
	var err error
	if v, err = wt.e.IsEnabled(ctx); err != nil {
		fatalf(wt.t, "IsEnabled: %s", err)
	}
	return
}

func (wt *webElementT) IsDisplayed(ctx context.Context) (v bool) {
	var err error
	if v, err = wt.e.IsDisplayed(ctx); err != nil {
		fatalf(wt.t, "IsDisplayed: %s", err)
	}
	return
}

func (wt *webElementT) GetAttribute(ctx context.Context, name string) (v string) {
	var err error
	if v, err = wt.e.GetAttribute(ctx, name); err != nil {
		fatalf(wt.t, "GetAttribute(%q): %s", name, err)
	}
	return
}

func (wt *webElementT) Location(ctx context.Context) (v *Point) {
	var err error
	if v, err = wt.e.Location(ctx); err != nil {
		fatalf(wt.t, "Location: %s", err)
	}
	return
}

func (wt *webElementT) LocationInView(ctx context.Context) (v *Point) {
	var err error
	if v, err = wt.e.LocationInView(ctx); err != nil {
		fatalf(wt.t, "LocationInView: %s", err)
	}
	return
}

func (wt *webElementT) Size(ctx context.Context) (v *Size) {
	var err error
	if v, err = wt.e.Size(ctx); err != nil {
		fatalf(wt.t, "Size: %s", err)
	}
	return
}

func (wt *webElementT) CSSProperty(ctx context.Context, name string) (v string) {
	var err error
	if v, err = wt.e.CSSProperty(ctx, name); err != nil {
		fatalf(wt.t, "CSSProperty(%q): %s", name, err)
	}
	return
}

func fatalf(t TestingT, fmtStr string, v ...interface{}) {
	// Backspace (delete) the file and line that t.Fatalf will add
	// that points to *this* invocation and replace it with that of
	// invocation of the webDriverT/webElementT method.
	_, thisFile, thisLine, _ := runtime.Caller(1)
	undoThisPrefix := strings.Repeat("\x08", len(fmt.Sprintf("%s:%d: ", filepath.Base(thisFile), thisLine)))
	_, file, line, _ := runtime.Caller(5)
	t.Fatalf(undoThisPrefix+filepath.Base(file)+":"+strconv.Itoa(line)+": "+fmtStr, v...)
}
