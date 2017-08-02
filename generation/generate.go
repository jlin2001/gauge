// Copyright 2015 ThoughtWorks, Inc.

// This file is part of Gauge.

// Gauge is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Gauge is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with Gauge.  If not, see <http://www.gnu.org/licenses/>.

package generation

import (
	"fmt"

	"github.com/getgauge/common"
	"github.com/getgauge/gauge/config"
	"github.com/getgauge/gauge/conn"
	"github.com/getgauge/gauge/gauge"
	"github.com/getgauge/gauge/gauge_messages"
	"github.com/getgauge/gauge/runner"
	"github.com/getgauge/gauge/validation"
)

func Generate(args []string) {
	if len(args) == 0 {
		args = append(args, common.SpecsDirectoryName)
	}
	result := validation.ValidateSpecs(args, false)
	for step := range result.ErrMap.StepErrs {
		generateStep(step, result.Runner)

	}
	result.Runner.Kill()

}

func generateStep(s *gauge.Step, r runner.Runner) {
	m := &gauge_messages.Message{MessageType: gauge_messages.Message_StepGenerateRequest, StepGenerateRequest: &gauge_messages.StepGenerateRequest{
		Step: &gauge_messages.ProtoStep{ActualText: s.LineText},
	}}
	res, err := getResponseFromRunner(m, r)
	if err != nil {
		fmt.Println(err.Error())
	}
	if res.GetMessageType() == gauge_messages.Message_StepGenerateResponse {
		res := res.GetStepGenerateResponse()
		fmt.Println(res)
	}
}

var getResponseFromRunner = func(m *gauge_messages.Message, r runner.Runner) (*gauge_messages.Message, error) {
	return conn.GetResponseForMessageWithTimeout(m, r.Connection(), config.RunnerRequestTimeout())
}
