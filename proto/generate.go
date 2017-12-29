// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// The proto package exists to generate the proto stubs.
package main

//go:generate rm ./go/shenzhen-go.pb.go
//go:generate rm ./js/shenzhen-go.pb.gopherjs.go
//go:generate protoc -I. shenzhen-go.proto --go_out=plugins=grpc:./go --gopherjs_out=plugins=grpc:./js

func main() {}
