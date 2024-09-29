package main

import (
	"reflect"
	"testing"
)

func TestDequeue(t *testing.T) {
	tests := map[string]struct {
		q              *queue
		removeMember   *Unacknowledged
		wantErr        bool
		wantReturn     *Unacknowledged
		wantQueueState []*Unacknowledged
	}{
		"dequeue empty queue": {
			q: &queue{
				queue: []*Unacknowledged{},
			},
			removeMember: &Unacknowledged{
				message: 0, neighbor: "n1",
			},
			wantErr:        true,
			wantReturn:     nil,
			wantQueueState: nil,
		},
		"dequeue empty queue implicitly": {
			q: &queue{
				queue: []*Unacknowledged{},
			},
			removeMember:   nil,
			wantErr:        true,
			wantReturn:     nil,
			wantQueueState: nil,
		},
		"dequeue nonexistent member": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
				},
			},
			removeMember: &Unacknowledged{
				message: 1, neighbor: "n2",
			},
			wantErr:        true,
			wantReturn:     nil,
			wantQueueState: nil,
		},
		"dequeue first member explicitly": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
				},
			},
			removeMember: &Unacknowledged{
				message: 0, neighbor: "n1",
			},
			wantErr: false,
			wantReturn: &Unacknowledged{
				message: 0, neighbor: "n1",
			},
			wantQueueState: []*Unacknowledged{
				{message: 1, neighbor: "n2"},
			},
		},
		"dequeue first member implicitly": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
				},
			},
			removeMember: nil,
			wantErr:      false,
			wantReturn: &Unacknowledged{
				message: 0, neighbor: "n1",
			},
			wantQueueState: []*Unacknowledged{
				{message: 1, neighbor: "n2"},
			},
		},
		"dequeue member in the middle": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
					{message: 2, neighbor: "n3"},
					{message: 3, neighbor: "n4"},
				},
			},
			removeMember: &Unacknowledged{
				message: 2, neighbor: "n3",
			},
			wantErr: false,
			wantReturn: &Unacknowledged{
				message: 2, neighbor: "n3",
			},
			wantQueueState: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
				{message: 1, neighbor: "n2"},
				{message: 3, neighbor: "n4"},
			},
		},
		"dequeue last member": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
					{message: 2, neighbor: "n3"},
					{message: 3, neighbor: "n4"},
				},
			},
			removeMember: &Unacknowledged{
				message: 3, neighbor: "n4",
			},
			wantErr: false,
			wantReturn: &Unacknowledged{
				message: 3, neighbor: "n4",
			},
			wantQueueState: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
				{message: 1, neighbor: "n2"},
				{message: 2, neighbor: "n3"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.q.Dequeue(tc.removeMember)
			if tc.wantErr && err != nil {
				return
			} else if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			} else if err != nil {
				t.Fatalf("unexpected error %v", err)
			}
			// check the return value
			if !reflect.DeepEqual(tc.wantReturn, got) {
				t.Fatalf("expected: %v, got: %v", tc.wantReturn, got)
			}
			// check the state of the queue
			for i, u := range tc.wantQueueState {
				if u.message != tc.q.queue[i].message && u.neighbor != tc.q.queue[i].neighbor {
					t.Fatalf("unexpected queue state, queue: %v, want: %v", tc.q.queue, tc.wantQueueState)
				}
			}
		})
	}
}

func TestEnqueue(t *testing.T) {
	tests := map[string]struct {
		q          *queue
		newMembers []*Unacknowledged
		want       []*Unacknowledged
	}{
		"enqueue on empty queue": {
			q: &queue{
				queue: []*Unacknowledged{},
			},
			newMembers: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
			},
			want: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
			},
		},
		"enqueue on existing queue": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
				},
			},
			newMembers: []*Unacknowledged{
				{message: 1, neighbor: "n2"},
			},
			want: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
				{message: 1, neighbor: "n2"},
			},
		},
		"order is preserved": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
					{message: 2, neighbor: "n3"},
					{message: 3, neighbor: "n4"},
					{message: 4, neighbor: "n5"},
				},
			},
			newMembers: []*Unacknowledged{
				{message: 5, neighbor: "n6"},
				{message: 6, neighbor: "n7"},
			},
			want: []*Unacknowledged{
				{message: 0, neighbor: "n1"},
				{message: 1, neighbor: "n2"},
				{message: 2, neighbor: "n3"},
				{message: 3, neighbor: "n4"},
				{message: 4, neighbor: "n5"},
				{message: 5, neighbor: "n6"},
				{message: 6, neighbor: "n7"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			for _, nm := range tc.newMembers {
				tc.q.Enqueue(nm)
			}
			if !reflect.DeepEqual(tc.want, tc.q.queue) {
				t.Fatalf("expected: %v, got: %v", tc.want, tc.q.queue)
			}
		})
	}
}

func TestIsEmpty(t *testing.T) {
	tests := map[string]struct {
		q    *queue
		want bool
	}{
		"queue with members": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
				},
			},
			want: false,
		},
		"queue without members": {
			q: &queue{
				queue: []*Unacknowledged{},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.q.IsEmpty()
			if got != tc.want {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}

func TestPeek(t *testing.T) {
	tests := map[string]struct {
		q       *queue
		wantErr bool
		want    *Unacknowledged
	}{
		"queue with members": {
			q: &queue{
				queue: []*Unacknowledged{
					{message: 0, neighbor: "n1"},
					{message: 1, neighbor: "n2"},
				},
			},
			wantErr: false,
			want:    &Unacknowledged{message: 0, neighbor: "n1"},
		},
		"queue without members": {
			q: &queue{
				queue: []*Unacknowledged{},
			},
			wantErr: true,
			want:    nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.q.Peek()
			if tc.wantErr && err != nil {
				return
			} else if tc.wantErr && err == nil {
				t.Fatalf("expected error")
			} else if err != nil {
				t.Fatalf("unexpected error %v", err)
			}

			if !reflect.DeepEqual(tc.want, got) {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}
