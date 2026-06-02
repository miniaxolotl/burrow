package internal

import (
	"fmt"

	stripeapi "github.com/stripe/stripe-go/v82"
	billingportal "github.com/stripe/stripe-go/v82/billingportal/session"
	checkout "github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/webhook"
)

type StripeClient struct {
	secretKey      string
	webhookSecret  string
	proPriceID     string
}

func NewStripeClient(secretKey, webhookSecret, proPriceID string) *StripeClient {
	stripeapi.Key = secretKey
	return &StripeClient{
		secretKey:     secretKey,
		webhookSecret: webhookSecret,
		proPriceID:    proPriceID,
	}
}

func (c *StripeClient) CreateCheckoutSession(customerEmail string, userID int64, successURL, cancelURL string) (string, error) {
	params := &stripeapi.CheckoutSessionParams{
		CustomerEmail: stripeapi.String(customerEmail),
		LineItems: []*stripeapi.CheckoutSessionLineItemParams{
			{
				Price:    stripeapi.String(c.proPriceID),
				Quantity: stripeapi.Int64(1),
			},
		},
		Mode:       stripeapi.String(string(stripeapi.CheckoutSessionModeSubscription)),
		SuccessURL: stripeapi.String(successURL),
		CancelURL:  stripeapi.String(cancelURL),
		Metadata: map[string]string{
			"user_id": fmt.Sprintf("%d", userID),
		},
	}
	s, err := checkout.New(params)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return s.URL, nil
}

func (c *StripeClient) CreatePortalSession(customerID, returnURL string) (string, error) {
	params := &stripeapi.BillingPortalSessionParams{
		Customer:  stripeapi.String(customerID),
		ReturnURL: stripeapi.String(returnURL),
	}
	s, err := billingportal.New(params)
	if err != nil {
		return "", fmt.Errorf("create portal session: %w", err)
	}
	return s.URL, nil
}

func (c *StripeClient) ConstructEvent(payload []byte, sig string) (stripeapi.Event, error) {
	return webhook.ConstructEvent(payload, sig, c.webhookSecret)
}
