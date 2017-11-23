package selenium

import (
	"context"
	"io"
)

/* Element finding options */
const (
	ById              = "id"
	ByXPATH           = "xpath"
	ByLinkText        = "link text"
	ByPartialLinkText = "partial link text"
	ByName            = "name"
	ByTagName         = "tag name"
	ByClassName       = "class name"
	ByCSSSelector     = "css selector"
)

/* Mouse buttons */
const (
	LeftButton = iota
	MiddleButton
	RightButton
)

/* Keys */
const (
	NullKey       = string('\ue000')
	CancelKey     = string('\ue001')
	HelpKey       = string('\ue002')
	BackspaceKey  = string('\ue003')
	TabKey        = string('\ue004')
	ClearKey      = string('\ue005')
	ReturnKey     = string('\ue006')
	EnterKey      = string('\ue007')
	ShiftKey      = string('\ue008')
	ControlKey    = string('\ue009')
	AltKey        = string('\ue00a')
	PauseKey      = string('\ue00b')
	EscapeKey     = string('\ue00c')
	SpaceKey      = string('\ue00d')
	PageUpKey     = string('\ue00e')
	PageDownKey   = string('\ue00f')
	EndKey        = string('\ue010')
	HomeKey       = string('\ue011')
	LeftArrowKey  = string('\ue012')
	UpArrowKey    = string('\ue013')
	RightArrowKey = string('\ue014')
	DownArrowKey  = string('\ue015')
	InsertKey     = string('\ue016')
	DeleteKey     = string('\ue017')
	SemicolonKey  = string('\ue018')
	EqualsKey     = string('\ue019')
	Numpad0Key    = string('\ue01a')
	Numpad1Key    = string('\ue01b')
	Numpad2Key    = string('\ue01c')
	Numpad3Key    = string('\ue01d')
	Numpad4Key    = string('\ue01e')
	Numpad5Key    = string('\ue01f')
	Numpad6Key    = string('\ue020')
	Numpad7Key    = string('\ue021')
	Numpad8Key    = string('\ue022')
	Numpad9Key    = string('\ue023')
	MultiplyKey   = string('\ue024')
	AddKey        = string('\ue025')
	SeparatorKey  = string('\ue026')
	SubstractKey  = string('\ue027')
	DecimalKey    = string('\ue028')
	DivideKey     = string('\ue029')
	F1Key         = string('\ue031')
	F2Key         = string('\ue032')
	F3Key         = string('\ue033')
	F4Key         = string('\ue034')
	F5Key         = string('\ue035')
	F6Key         = string('\ue036')
	F7Key         = string('\ue037')
	F8Key         = string('\ue038')
	F9Key         = string('\ue039')
	F10Key        = string('\ue03a')
	F11Key        = string('\ue03b')
	F12Key        = string('\ue03c')
	MetaKey       = string('\ue03d')
)

/* Browser capabilities, see
http://code.google.com/p/selenium/wiki/JsonWireProtocol#Capabilities_JSON_Object
*/
type Capabilities map[string]interface{}

/* Build object, part of Status return. */
type Build struct {
	Version, Revision, Time string
}

/* OS object, part of Status return. */
type OS struct {
	Arch, Name, Version string
}

/* Information retured by Status method. */
type Status struct {
	Build `json:"build"`
	OS    `json:"os"`
}

/* Point */
type Point struct {
	X, Y float64
}

/* Size */
type Size struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

/* Cookie */
type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Path   string `json:"path"`
	Domain string `json:"domain"`
	Secure bool   `json:"secure"`
	Expiry uint   `json:"-"`
}

type WebDriver interface {

	/* Status (info) on server */
	Status(ctx context.Context) (*Status, error)

	/* List of actions on the server. */
	Sessions(ctx context.Context) ([]Session, error)

	/* Start a new session, return session id */
	NewSession(ctx context.Context) (string, error)

	/* Return the current session ID */
	GetSessionID() string

	/* Current session capabilities */
	Capabilities(ctx context.Context) (Capabilities, error)

	/* Configure the amount of time a particular type of operation can execute for before it is aborted.
	   Valid types: "script" for script timeouts, "implicit" for modifying the implicit wait timeout and "page load" for setting a page load timeout. */
	SetTimeout(ctx context.Context, timeoutType string, ms uint) error
	/* Set the amount of time, in milliseconds, that asynchronous scripts are permitted to run before they are aborted. */
	SetAsyncScriptTimeout(ctx context.Context, ms uint) error
	/* Set the amount of time, in milliseconds, the driver should wait when searching for elements. */
	SetImplicitWaitTimeout(ctx context.Context, ms uint) error

	// IME
	/* List all available engines on the machine. */
	AvailableEngines(ctx context.Context) ([]string, error)
	/* Get the name of the active IME engine. */
	ActiveEngine(ctx context.Context) (string, error)
	/* Indicates whether IME input is active at the moment. */
	IsEngineActivated(ctx context.Context) (bool, error)
	/* De-activates the currently-active IME engine. */
	DeactivateEngine(ctx context.Context) error
	/* Make an engines active */
	ActivateEngine(ctx context.Context, engine string) error

	/* Quit (end) current session */
	Quit(ctx context.Context) error

	// Page information and manipulation
	/* Return id of current window handle. */
	CurrentWindowHandle(ctx context.Context) (string, error)
	/* Return ids of current open windows. */
	WindowHandles(ctx context.Context) ([]string, error)
	/* Current url. */
	CurrentURL(ctx context.Context) (string, error)
	/* Page title. */
	Title(ctx context.Context) (string, error)
	/* Get page source. */
	PageSource(ctx context.Context) (string, error)
	/* Close current window. */
	Close(ctx context.Context) error
	/* Switch to frame, frame parameter can be name or id. */
	SwitchFrame(ctx context.Context, frame string) error
	/* Switch to parent frame */
	SwitchFrameParent(ctx context.Context) error
	/* Swtich to window. */
	SwitchWindow(ctx context.Context, name string) error
	/* Close window. */
	CloseWindow(ctx context.Context, name string) error
	/* Get window size */
	WindowSize(ctx context.Context, name string) (*Size, error)
	/* Get window position */
	WindowPosition(ctx context.Context, name string) (*Point, error)

	// ResizeWindow resizes the named window.
	ResizeWindow(ctx context.Context, name string, to Size) error

	// Navigation
	/* Open url. */
	Get(ctx context.Context, url string) error
	/* Move forward in history. */
	Forward(ctx context.Context) error
	/* Move backward in history. */
	Back(ctx context.Context) error
	/* Refresh page. */
	Refresh(ctx context.Context) error

	// Finding element(s)
	/* Find, return one element. */
	FindElement(ctx context.Context, by, value string) (WebElement, error)
	/* Find, return list of elements. */
	FindElements(ctx context.Context, by, value string) ([]WebElement, error)
	/* Current active element. */
	ActiveElement(ctx context.Context) (WebElement, error)

	// Shortcut for FindElement(ByCSSSelector, sel)
	Q(ctx context.Context, sel string) (WebElement, error)
	// Shortcut for FindElements(ByCSSSelector, sel)
	QAll(ctx context.Context, sel string) ([]WebElement, error)

	// Cookies
	/* Get all cookies */
	GetCookies(ctx context.Context) ([]Cookie, error)
	/* Add a cookie */
	AddCookie(ctx context.Context, cookie *Cookie) error
	/* Delete all cookies */
	DeleteAllCookies(ctx context.Context) error
	/* Delete a cookie */
	DeleteCookie(ctx context.Context, name string) error

	// Mouse
	/* Click mouse button, button should be on of RightButton, MiddleButton or
	LeftButton.
	*/
	Click(ctx context.Context, button int) error
	/* Dobule click */
	DoubleClick(ctx context.Context) error
	/* Mouse button down */
	ButtonDown(ctx context.Context) error
	/* Mouse button up */
	ButtonUp(ctx context.Context) error

	// Misc
	/* Send modifier key to active element.
	modifier can be one of ShiftKey, ControlKey, AltKey, MetaKey.
	*/
	SendModifier(ctx context.Context, modifier string, isDown bool) error
	Screenshot(ctx context.Context) (io.Reader, error)

	// Alerts
	/* Dismiss current alert. */
	DismissAlert(ctx context.Context) error
	/* Accept current alert. */
	AcceptAlert(ctx context.Context) error
	/* Current alert text. */
	AlertText(ctx context.Context) (string, error)
	/* Set current alert text. */
	SetAlertText(ctx context.Context, text string) error

	// Scripts
	/* Execute a script. */
	ExecuteScript(ctx context.Context, script string, args []interface{}) (interface{}, error)
	/* Execute a script async. */
	ExecuteScriptAsync(ctx context.Context, script string, args []interface{}) (interface{}, error)

	// Get a WebDriverT of this element that has methods that call t.Fatalf upon
	// encountering errors instead of using multiple returns to indicate errors.
	// The argument t is typically a *testing.T, but here it's a similar
	// interface to avoid needing to import "testing" (which registers global
	// command-line flags).
	T(t TestingT) WebDriverT

	// Raw execution
	VoidExecute(ctx context.Context, url string, params interface{}) error
}

type WebElement interface {
	// Manipulation

	/* Click on element */
	Click(ctx context.Context) error
	/* Send keys (type) into element */
	SendKeys(ctx context.Context, keys string) error
	/* Submit */
	Submit(ctx context.Context) error
	/* Clear */
	Clear(ctx context.Context) error
	/* Move mouse to relative coordinates */
	MoveTo(ctx context.Context, xOffset, yOffset int) error

	// Finding

	/* Find children, return one element. */
	FindElement(ctx context.Context, by, value string) (WebElement, error)
	/* Find children, return list of elements. */
	FindElements(ctx context.Context, by, value string) ([]WebElement, error)

	// Shortcut for FindElement(ByCSSSelector, sel)
	Q(ctx context.Context, sel string) (WebElement, error)
	// Shortcut for FindElements(ByCSSSelector, sel)
	QAll(ctx context.Context, sel string) ([]WebElement, error)

	// Porperties

	/* Element name */
	TagName(ctx context.Context) (string, error)
	/* Text of element */
	Text(ctx context.Context) (string, error)
	/* Check if element is selected. */
	IsSelected(ctx context.Context) (bool, error)
	/* Check if element is enabled. */
	IsEnabled(ctx context.Context) (bool, error)
	/* Check if element is displayed. */
	IsDisplayed(ctx context.Context) (bool, error)
	/* Get element attribute. */
	GetAttribute(ctx context.Context, name string) (string, error)
	/* Element location. */
	Location(ctx context.Context) (*Point, error)
	/* Element location once it has been scrolled into view.
	   Note: This is considered an internal command and should only be used to determine an element's location for correctly generating native events.*/
	LocationInView(ctx context.Context) (*Point, error)
	/* Element size */
	Size(ctx context.Context) (*Size, error)
	/* Get element CSS property value. */
	CSSProperty(ctx context.Context, name string) (string, error)

	// Get a WebElementT of this element that has methods that call t.Fatalf
	// upon encountering errors instead of using multiple returns to indicate
	// errors. The argument t is typically a *testing.T, but here it's a similar
	// interface to avoid needing to import "testing" (which registers global
	// command-line flags).
	T(t TestingT) WebElementT
}

// TestingT is a subset of the testing.T interface (to avoid needing
// to import "testing", which registers global command-line flags).
type TestingT interface {
	Fatalf(fmt string, v ...interface{})
}
