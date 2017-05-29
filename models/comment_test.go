package models

import (
	"reflect"
	"testing"
)

func TestNewComment(t *testing.T) {
	author, _ := NewUser("testuser1@example.com", "testuser1", "testpassword")
	a := NewArticle("title", "desc", "body", author)
	u, _ := NewUser("testuser@example.com", "testuser", "testpassword")

	type args struct {
		article *Article
		user    *User
		body    string
	}
	tests := []struct {
		name    string
		args    args
		want    *Comment
		wantErr bool
	}{
		{
			"get an error when body is empty",
			args{article: a, user: u, body: ""},
			nil, true,
		},
		{
			"initialize a new comment",
			args{article: a, user: u, body: "A comment body"},
			&Comment{Article: *a, User: *u, Body: "A comment body"},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewComment(tt.args.article, tt.args.user, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewComment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewComment() = %v, want %v", got, tt.want)
			}
		})
	}
}
