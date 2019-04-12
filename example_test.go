// Copyright 2019 Mike Gerow
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

package pager_test

import (
	"fmt"

	"github.com/gerow/pager"
)

func ExampleOpen() {
	pager.Open()
	defer pager.Close()

	for i := 0; i < 10; i++ {
		fmt.Printf("%d hello from my pager!\n", i)
	}

	// Output:
	// 0 hello from my pager!
	// 1 hello from my pager!
	// 2 hello from my pager!
	// 3 hello from my pager!
	// 4 hello from my pager!
	// 5 hello from my pager!
	// 6 hello from my pager!
	// 7 hello from my pager!
	// 8 hello from my pager!
	// 9 hello from my pager!
}
