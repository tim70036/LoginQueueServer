package infra

import (
	"time"

	"github.com/imroc/req/v3"
)

var HttpClient = req.C(). // Use C() to create a client and set with chainable client settings.
	// Timeout of all requests.
	SetTimeout(10 * time.Second).
	// Enable retry and set the maximum retry count.
	SetCommonRetryCount(3).
	// Set the retry sleep interval with a commonly used algorithm: capped exponential backoff with jitter (https://aws.amazon.com/blogs/architecture/exponential-backoff-and-jitter/).
	SetCommonRetryFixedInterval(3 * time.Second).EnableDumpEachRequest()
