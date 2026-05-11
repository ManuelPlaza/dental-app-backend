package arangodb

import (
	"context"
	"fmt"
	"time"

	"dental-app/internal/core/domain"
	"dental-app/internal/core/ports"

	"github.com/arangodb/go-driver/v2/arangodb"
)

type arangoReferralRepo struct {
	db arangodb.Database
}

func NewReferralRepo(c *Client) ports.ReferralGraphRepository {
	return &arangoReferralRepo{db: c.DB}
}

func (r *arangoReferralRepo) SyncPatient(id uint, doc, name string) error {
	if r.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aql := `
		UPSERT { _key: @doc }
		INSERT { _key: @doc, document_number: @doc, name: @name, patient_id: @id }
		UPDATE { name: @name, patient_id: @id }
		IN patients
	`
	cursor, err := r.db.Query(ctx, aql, &arangodb.QueryOptions{
		BindVars: map[string]interface{}{"doc": doc, "name": name, "id": id},
	})
	if err != nil {
		return fmt.Errorf("arango sync patient: %w", err)
	}
	return cursor.Close()
}

func (r *arangoReferralRepo) LinkReferral(referrerDoc, referredDoc string) error {
	if r.db == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	from := "patients/" + referrerDoc
	to := "patients/" + referredDoc
	aql := `
		UPSERT { _from: @from, _to: @to }
		INSERT { _from: @from, _to: @to, created_at: DATE_ISO8601(DATE_NOW()) }
		UPDATE {}
		IN referred
	`
	cursor, err := r.db.Query(ctx, aql, &arangodb.QueryOptions{
		BindVars: map[string]interface{}{"from": from, "to": to},
	})
	if err != nil {
		return fmt.Errorf("arango link referral: %w", err)
	}
	return cursor.Close()
}

func (r *arangoReferralRepo) GetPatientReferralStatus(doc string) (*domain.PatientReferralStatus, error) {
	if r.db == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	aql := `
		LET doc = @doc
		LET out_count = LENGTH(
			FOR edge IN referred FILTER edge._from == CONCAT("patients/", doc) RETURN 1
		)
		LET referred_by_name = FIRST(
			FOR edge IN referred
			FILTER edge._to == CONCAT("patients/", doc)
			FOR p IN patients FILTER p._id == edge._from
			RETURN p.name
		)
		RETURN { referral_count: out_count, referred_by_name: referred_by_name }
	`
	cursor, err := r.db.Query(ctx, aql, &arangodb.QueryOptions{
		BindVars: map[string]interface{}{"doc": doc},
	})
	if err != nil {
		return nil, fmt.Errorf("arango referral status: %w", err)
	}
	defer cursor.Close()

	if !cursor.HasMore() {
		return &domain.PatientReferralStatus{}, nil
	}
	var status domain.PatientReferralStatus
	if _, err := cursor.ReadDocument(ctx, &status); err != nil {
		return nil, fmt.Errorf("arango referral status read: %w", err)
	}
	return &status, nil
}

func (r *arangoReferralRepo) TopReferrers(limit int) ([]domain.TopReferrer, error) {
	if r.db == nil {
		return []domain.TopReferrer{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	aql := `
		FOR referrer IN patients
		  LET count = LENGTH(FOR edge IN referred FILTER edge._from == referrer._id RETURN 1)
		  FILTER count > 0
		  SORT count DESC
		  LIMIT @limit
		  RETURN {
		    document_number: referrer.document_number,
		    name: referrer.name,
		    referral_count: count
		  }
	`
	cursor, err := r.db.Query(ctx, aql, &arangodb.QueryOptions{
		BindVars: map[string]interface{}{"limit": limit},
	})
	if err != nil {
		return nil, fmt.Errorf("arango top referrers: %w", err)
	}
	defer cursor.Close()

	var results []domain.TopReferrer
	for cursor.HasMore() {
		var item domain.TopReferrer
		if _, err := cursor.ReadDocument(ctx, &item); err != nil {
			break
		}
		results = append(results, item)
	}
	if results == nil {
		results = []domain.TopReferrer{}
	}
	return results, nil
}
