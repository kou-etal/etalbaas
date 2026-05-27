package handler

import (
	secretv1 "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func SecretMetadataToProto(row store.SecretsMetadatum) *secretv1.SecretMetadata {
	return &secretv1.SecretMetadata{
		Id:          row.ID.String(),
		ProjectId:   row.ProjectID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   timestamppb.New(row.CreatedAt),
		UpdatedAt:   timestamppb.New(row.UpdatedAt),
	}
}

func SecretsToProto(rows []store.SecretsMetadatum, hasMore bool) ([]*secretv1.SecretMetadata, string) {
	secrets := make([]*secretv1.SecretMetadata, len(rows))
	for i, r := range rows {
		secrets[i] = SecretMetadataToProto(r)
	}
	var nextToken string
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		nextToken = encodeUUIDCursor(last.CreatedAt, last.ID)
	}
	return secrets, nextToken
}
