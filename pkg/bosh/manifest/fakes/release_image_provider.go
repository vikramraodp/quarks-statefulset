// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	sync "sync"

	manifest "code.cloudfoundry.org/cf-operator/pkg/bosh/manifest"
)

type FakeReleaseImageProvider struct {
	GetReleaseImageStub        func(string, string) (string, error)
	getReleaseImageMutex       sync.RWMutex
	getReleaseImageArgsForCall []struct {
		arg1 string
		arg2 string
	}
	getReleaseImageReturns struct {
		result1 string
		result2 error
	}
	getReleaseImageReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeReleaseImageProvider) GetReleaseImage(arg1 string, arg2 string) (string, error) {
	fake.getReleaseImageMutex.Lock()
	ret, specificReturn := fake.getReleaseImageReturnsOnCall[len(fake.getReleaseImageArgsForCall)]
	fake.getReleaseImageArgsForCall = append(fake.getReleaseImageArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("GetReleaseImage", []interface{}{arg1, arg2})
	fake.getReleaseImageMutex.Unlock()
	if fake.GetReleaseImageStub != nil {
		return fake.GetReleaseImageStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getReleaseImageReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeReleaseImageProvider) GetReleaseImageCallCount() int {
	fake.getReleaseImageMutex.RLock()
	defer fake.getReleaseImageMutex.RUnlock()
	return len(fake.getReleaseImageArgsForCall)
}

func (fake *FakeReleaseImageProvider) GetReleaseImageCalls(stub func(string, string) (string, error)) {
	fake.getReleaseImageMutex.Lock()
	defer fake.getReleaseImageMutex.Unlock()
	fake.GetReleaseImageStub = stub
}

func (fake *FakeReleaseImageProvider) GetReleaseImageArgsForCall(i int) (string, string) {
	fake.getReleaseImageMutex.RLock()
	defer fake.getReleaseImageMutex.RUnlock()
	argsForCall := fake.getReleaseImageArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeReleaseImageProvider) GetReleaseImageReturns(result1 string, result2 error) {
	fake.getReleaseImageMutex.Lock()
	defer fake.getReleaseImageMutex.Unlock()
	fake.GetReleaseImageStub = nil
	fake.getReleaseImageReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeReleaseImageProvider) GetReleaseImageReturnsOnCall(i int, result1 string, result2 error) {
	fake.getReleaseImageMutex.Lock()
	defer fake.getReleaseImageMutex.Unlock()
	fake.GetReleaseImageStub = nil
	if fake.getReleaseImageReturnsOnCall == nil {
		fake.getReleaseImageReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.getReleaseImageReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeReleaseImageProvider) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.getReleaseImageMutex.RLock()
	defer fake.getReleaseImageMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeReleaseImageProvider) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ manifest.ReleaseImageProvider = new(FakeReleaseImageProvider)
