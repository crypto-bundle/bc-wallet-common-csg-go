/*
 * MIT NON-AI License
 *
 * Copyright (c) 2022-2025 Aleksei Kotelnikov(gudron2s@gmail.com)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of the software and associated documentation files (the "Software"),
 * to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense,
 * and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions.
 *
 * The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
 *
 * In addition, the following restrictions apply:
 *
 * 1. The Software and any modifications made to it may not be used for the purpose of training or improving machine learning algorithms,
 * including but not limited to artificial intelligence, natural language processing, or data mining. This condition applies to any derivatives,
 * modifications, or updates based on the Software code. Any usage of the Software in an AI-training dataset is considered a breach of this License.
 *
 * 2. The Software may not be included in any dataset used for training or improving machine learning algorithms,
 * including but not limited to artificial intelligence, natural language processing, or data mining.
 *
 * 3. Any person or organization found to be in violation of these restrictions will be subject to legal action and may be held liable
 * for any damages resulting from such use.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
 * DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE
 * OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 *
 */

package csg

import "errors"

var _ errorFormatterService = (*mockErrFmt)(nil)

// ATTENTION: DO NOT USE THIS ERROR FORMATTER IN PRODUCTION CODE.
//
// errMockFormatterSvc - internal implementation of standard crypto-bundle error formatter service,
// only for mock scenario...
var errMockFormatterSvc = errors.New("mock_err_formatter")

type mockErrFmt struct {
}

func (f *mockErrFmt) NewErrorWithCode(text string, code int) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) ErrorCodeIsOneOf(err error, codes ...int) (int, bool) {
	return -1, false
}

func (f *mockErrFmt) ErrCodeIsOneOf(err error, codes ...int) (int, bool) {
	return -1, false
}

func (f *mockErrFmt) ErrorWithCode(_ error, _ int) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) ErrWithCode(_ error, _ int) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) ErrorGetCode(_ error) int {
	return -1
}

func (f *mockErrFmt) ErrGetCode(_ error) int {
	return -1
}

func (f *mockErrFmt) ErrorNoWrap(_ error) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) ErrNoWrap(_ error) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) ErrorOnly(_ error, _ ...string) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) Error(_ error, _ ...string) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) Errorf(_ error, _ string, _ ...interface{}) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) NewError(_ ...string) error {
	return errMockFormatterSvc
}

func (f *mockErrFmt) NewErrorf(_ string, _ ...interface{}) error {
	return errMockFormatterSvc
}

func NewMockErrFormatter() *mockErrFmt {
	return &mockErrFmt{}
}
