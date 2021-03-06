package response

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"

	"github.com/aphistic/sweet"
	. "github.com/onsi/gomega"
)

type ResponseSuite struct{}

func (s *ResponseSuite) TestSetters(t sweet.T) {
	resp := newResponse(nil)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(resp.SetStatusCode(http.StatusNotFound)).To(Equal(resp))
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	Expect(resp.SetHeader("X-Foo", "bar")).To(Equal(resp))
	Expect(resp.GetHeader("X-Foo")).To(Equal("bar"))
	Expect(resp.SetHeader("X-Foo", "baz")).To(Equal(resp))
	Expect(resp.GetHeader("X-Foo")).To(Equal("baz"))

	Expect(resp.AddHeader("X-Foo", "bonk")).To(Equal(resp))
	Expect(resp.GetHeader("X-Foo")).To(Equal("baz"))
	Expect(resp.Header["X-Foo"]).To(Equal([]string{"baz", "bonk"}))
}

func (s *ResponseSuite) TestConvert(t sweet.T) {
	var (
		errors = make(chan error, 2)
		c1     = func(err error) { errors <- err }
		c2     = func(err error) { errors <- err }
	)

	server := httptest.NewServer(Convert(func(r *http.Request) *Response {
		defer r.Body.Close()
		data, _ := ioutil.ReadAll(r.Body)

		resp := JSON(map[string]interface{}{"input": string(data)})
		resp.SetStatusCode(http.StatusAccepted)
		resp.AddHeader("X-Context", "test")
		resp.AddCallback(c1)
		resp.AddCallback(c2)
		return resp
	}))

	defer close(errors)
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, bytes.NewReader([]byte("content")))
	resp, err := http.DefaultClient.Do(req)
	Expect(err).To(BeNil())

	Eventually(errors).Should(Receive(nil))
	Eventually(errors).Should(Receive(nil))
	Consistently(errors).ShouldNot(Receive())

	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	Expect(resp.StatusCode).To(Equal(http.StatusAccepted))
	Expect(resp.Header.Get("X-Context")).To(Equal("test"))
	Expect(resp.Header.Get("Content-Type")).To(Equal("application/json"))
	Expect(data).To(MatchJSON(`{"input": "content"}`))
}

func (s *ResponseSuite) TestMultipleWriteToCallsPanics(t sweet.T) {
	resp := JSON(nil)

	// This one is fine
	resp.WriteTo(httptest.NewRecorder())

	// This one is not
	Expect(func() { resp.WriteTo(httptest.NewRecorder()) }).To(Panic())
}
