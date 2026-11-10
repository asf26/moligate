/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"testing"

	dto "github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/stretchr/testify/assert"
)

func TestTaskStatusToVideoStatus(t *testing.T) {
	// A task row starts as NOT_START right after submit; callers poll immediately,
	// so that state must read as queued rather than unknown.
	assert.Equal(t, dto.VideoStatusQueued, TaskStatusNotStart.ToVideoStatus())
	assert.Equal(t, dto.VideoStatusQueued, TaskStatus(TaskStatusQueued).ToVideoStatus())
	assert.Equal(t, dto.VideoStatusQueued, TaskStatus(TaskStatusSubmitted).ToVideoStatus())
	assert.Equal(t, dto.VideoStatusInProgress, TaskStatus(TaskStatusInProgress).ToVideoStatus())
	assert.Equal(t, dto.VideoStatusCompleted, TaskStatus(TaskStatusSuccess).ToVideoStatus())
	assert.Equal(t, dto.VideoStatusFailed, TaskStatus(TaskStatusFailure).ToVideoStatus())
}
