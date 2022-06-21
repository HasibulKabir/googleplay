package googleplay

import (
   "errors"
   "github.com/89z/format"
   "github.com/89z/format/crypto"
   "github.com/89z/format/net"
   "net/http"
   "net/url"
   "os"
   "strconv"
   "strings"
   "time"
)

const Sleep = 4 * time.Second

var Log format.Log

func (t Token) Create(name string) error {
   file, err := format.Create(name)
   if err != nil {
      return err
   }
   defer file.Close()
   return net.Encode(file, t.Values)
}

func Open_Token(name string) (*Token, error) {
   file, err := os.Open(name)
   if err != nil {
      return nil, err
   }
   defer file.Close()
   var tok Token
   tok.Values, err = net.Decode(file)
   if err != nil {
      return nil, err
   }
   return &tok, nil
}

func (t Token) Header(device_ID uint64, single bool) (*Header, error) {
   // these values take from Android API 28
   body := url.Values{
      "Token": {t.Token()},
      "app": {"com.android.vending"},
      "client_sig": {"38918a453d07199354f8b19af05ec6562ced5788"},
      "service": {"oauth2:https://www.googleapis.com/auth/googleplay"},
   }.Encode()
   req, err := http.NewRequest(
      "POST", "https://android.googleapis.com/auth", strings.NewReader(body),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   Log.Dump(req)
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return nil, errors.New(res.Status)
   }
   var head Header
   head.SDK = 9
   head.Device_ID = device_ID
   if single {
      head.Version_Code = 8091_9999 // single APK
   } else {
      head.Version_Code = 9999_9999
   }
   val, err := net.Decode(res.Body)
   if err != nil {
      return nil, err
   }
   head.Auth = val.Get("Auth")
   return &head, nil
}

type Header struct {
   Device_ID uint64 // X-DFE-Device-ID
   SDK int64 // User-Agent
   Version_Code int64 // User-Agent
   Auth string // Authorization
}

func (h Header) Set_Agent(head http.Header) {
   var buf []byte
   buf = append(buf, "Android-Finsky (sdk="...)
   buf = strconv.AppendInt(buf, h.SDK, 10)
   buf = append(buf, ",versionCode="...)
   buf = strconv.AppendInt(buf, h.Version_Code, 10)
   buf = append(buf, ')')
   head.Set("User-Agent", string(buf))
}

func (h Header) Set_Auth(head http.Header) {
   head.Set("Authorization", "Bearer " + h.Auth)
}

func (h Header) Set_Device(head http.Header) {
   device := strconv.FormatUint(h.Device_ID, 16)
   head.Set("X-DFE-Device-ID", device)
}

// Purchase app. Only needs to be done once per Google account.
func (h Header) Purchase(app string) error {
   query := "doc=" + url.QueryEscape(app)
   req, err := http.NewRequest(
      "POST", "https://android.clients.google.com/fdfe/purchase",
      strings.NewReader(query),
   )
   if err != nil {
      return err
   }
   h.Set_Auth(req.Header)
   h.Set_Device(req.Header)
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   Log.Dump(req)
   res, err := new(http.Transport).RoundTrip(req)
   if err != nil {
      return err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return errors.New(res.Status)
   }
   return nil
}

func (t Token) Token() string {
   return t.Get("Token")
}

// You can also use host "android.clients.google.com", but it also uses
// TLS fingerprinting.
func New_Token(email, password string) (*Token, error) {
   body := url.Values{
      "Email": {email},
      "Passwd": {password},
      "client_sig": {""},
      "droidguard_results": {""},
   }.Encode()
   req, err := http.NewRequest(
      "POST", "https://android.googleapis.com/auth", strings.NewReader(body),
   )
   if err != nil {
      return nil, err
   }
   req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
   hello, err := crypto.Parse_JA3(crypto.Android_API_26)
   if err != nil {
      return nil, err
   }
   Log.Dump(req)
   res, err := crypto.Transport(hello).RoundTrip(req)
   if err != nil {
      return nil, err
   }
   defer res.Body.Close()
   if res.StatusCode != http.StatusOK {
      return nil, errors.New(res.Status)
   }
   var tok Token
   tok.Values, err = net.Decode(res.Body)
   if err != nil {
      return nil, err
   }
   return &tok, nil
}

type Token struct {
   url.Values
}
