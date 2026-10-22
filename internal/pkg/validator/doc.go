// Package validator registers application validation rules and formats request
// errors in Indonesian. New configures a reusable instance for validate tags;
// Register configures an existing instance, such as Gin's binding validator.
// Registration must finish before the instance serves concurrent requests.
//
// FormatError builds a transport-independent validation response.
// SendValidationError writes that response in the application's Gin JSON format.
// Meaningful-text detection is opt-in through the meaningful_text tag or
// IsMeaningfulText; AnalyzeMeaningfulText also returns classification and reasons.
// These functions do not establish requiredness or length limits.
package validator
