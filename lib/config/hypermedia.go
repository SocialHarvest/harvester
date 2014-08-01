package config

// Inspired by a few hypermedia formats, this is a structure for Social Harvest API responses.
// Storing data into Social Harvest is easy...Getting it back out and having other widgets for the dashboard be able to talk with the API is the hard part.
// So a self documenting API that can be navigated automatically is super handy.

// A resource is the root level item being returned. It can contain embedded resources if necessary. It's possible to return more than one resource at a time too (though won't be common).
// Within each resource there is "_meta" data
type HypermediaResource struct {
	Meta     HypermediaMeta                `json:"_meta"`
	Links    map[string]HypermediaLink     `json:"_links,omitempty"`
	Curies   map[string]HypermediaCurie    `json:"_curies,omitempty"`
	Data     map[string]interface{}        `json:"_data,omitempty"`
	Embedded map[string]HypermediaResource `json:"_embedded,omitempty"`
	Forms    map[string]HypermediaForm     `json:"_forms,omitempty"`
}

// The Meta structure provides some common information helpful to the application and also resource state.
type HypermediaMeta struct {
	Success      bool              `json:"success"`
	Message      string            `json:"message"`
	ResponseTime float32           `json:"responseTime"`
	DataCount    int               `json:"dataCount,omitempty"`
	DataLimit    int               `json:"dataLimit,omitempty"`
	DataTotal    int               `json:"dataTotal,omitempty"`
	DataOrder    map[string]string `json:"dataOrder,omitempty"`
}

// A simple web link structure (somewhat modeled after HAL's links and http://tools.ietf.org/html/rfc5988).
// NOTE: in HAL format, links can be an array with aliases - our format has no such support, but this doens't break HAL compatibility.
// Why not support it? Because that changes from {} to [] and changing data types is a burden for others. Plus we have HTTP 301/302.
// Also, each "_links" key name using this struct should be one of: http://www.iana.org/assignments/link-relations/link-relations.xhtml unless using CURIEs.
type HypermediaLink struct {
	Href        string `json:"href"`
	Type        string `json:"type,omitempty"`
	Deprecation string `json:"deprecation,omitempty"`
	Name        string `json:"name,omitempty"`
	Profile     string `json:"profile,omitempty"`
	Title       string `json:"title,omitempty"`
	Hreflang    string `json:"hreflang,omitempty"`
}

// Defines a CURIE
type HypermediaCurie struct {
	Name      string `json:"name,omitempty"`
	Href      string `json:"href,omitempty"`
	Templated bool   `json:"templated,omitempty"`
}

// Form structure defines attributes that match HTML. This tells applications how to work with resources.
// Any attribute not found in HTML should be prefixed with an underscore (for example, "_fields").
type HypermediaForm struct {
	Name          string                         `json:"name,omitempty"`
	Method        string                         `json:"method,omitempty"`
	Enctype       string                         `json:"enctype"`
	AcceptCharset string                         `json:"accept-charset,omitempty"`
	Target        string                         `json:"target,omitempty"`
	Action        string                         `json:"action,omitempty"`
	Autocomplete  bool                           `json:"autocomplete,omitempty"`
	Fields        map[string]HypermediaFormField `json:"_fields,omitempty"`
}

// Defines properties for a field (HTML attributes) as well as holds the "_errors" and validation "_rules" for that field.
// "_rules" have key names that map to HypermediaFormField.Name, like { "fieldName": HypermediaFormFieldRule } and the rules themself are named.
// "_errors" have key names that also map to HypermediaFormField.Name
type HypermediaFormField struct {
	Name         string                              `json:"name,omitempty"`
	Value        string                              `json:"value,omitempty"`
	Type         string                              `json:"type,omitempty"`
	Src          string                              `json:"src,omitempty"`
	Checked      bool                                `json:"checked,omitempty"`
	Disabled     bool                                `json:"disabled,omitempty"`
	ReadOnly     bool                                `json:"readonly,omitempty"`
	Required     bool                                `json:"required,omitempty"`
	Autocomplete bool                                `json:"autocomplete,omitempty"`
	Tabindex     int                                 `json:"tabindex,omitempty"`
	Multiple     bool                                `json:"multiple,omitempty"`
	Accept       string                              `json:"accept,omitempty"`
	Errors       map[string]HypermediaFormFieldError `json:"_errors,omitempty"`
	Rules        map[string]HypermediaFormFieldRule  `json:"_rules,omitempty"`
}

// Error messages from validation failures (optional) "name" is the HypermediaFormFieldRule.Name in this case and "message" is returned on failure.
type HypermediaFormFieldError struct {
	Name    string `json:"name"`
	Failed  bool   `json:"name"`
	Message string `json:"message,omitempty"`
}

// Simple validation rules. Easily nested into "_rules" on "_fields" (optional). Of course front-end validation is merely convenience and not a trustable process.
// So remember to sanitize and validate any data on the server side of things. However, this does help tremendously in reducing the number of HTTP requests to the API.
type HypermediaFormFieldRule struct {
	Name        string                                         `json:"name"`
	Description string                                         `json:"description,omitempty"`
	Pattern     string                                         `json:"pattern"`
	Function    func(value string) (fail bool, message string) // not for JSON
}

// Example
// var hyper = config.HypermediaResource{}
// //var selfLink = config.HypermediaLink{Href: "http://www.google.com"}
// hyper.Links = make(map[string]config.HypermediaLink)
// hyper.Links["self"] = config.HypermediaLink{Href: "http://www.google.com"}
// // now embed another resource within it
// var hEmbedded = config.HypermediaResource{}
// hEmbedded.Links = make(map[string]config.HypermediaLink)
// hEmbedded.Links["self"] = config.HypermediaLink{Href: "http://www.embedded.com"}
// //log.Println(hEmbedded)
// hyper.Embedded = make(map[string]config.HypermediaResource)
// hyper.Embedded["testEmbeddedResource"] = hEmbedded
// hJson, _ := json.Marshal(hyper)
// log.Println(string(hJson))
