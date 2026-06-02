package internal

import (
	"context"
	"encoding/json"
	"fmt"

	stripeapi "github.com/stripe/stripe-go/v82"
)

func CreateCheckout(ctx context.Context, stripeClient *StripeClient, pg *PostgresClient, userID int64, successURL, cancelURL string) (string, error) {
	user, err := pg.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	url, err := stripeClient.CreateCheckoutSession(user.Email, user.ID, successURL, cancelURL)
	if err != nil {
		return "", fmt.Errorf("create checkout: %w", err)
	}
	return url, nil
}

func CreatePortal(ctx context.Context, stripeClient *StripeClient, pg *PostgresClient, userID int64, returnURL string) (string, error) {
	user, err := pg.GetUserByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		return "", fmt.Errorf("no stripe customer id")
	}
	url, err := stripeClient.CreatePortalSession(*user.StripeCustomerID, returnURL)
	if err != nil {
		return "", fmt.Errorf("create portal: %w", err)
	}
	return url, nil
}

func ProcessWebhookEvent(ctx context.Context, stripeClient *StripeClient, pg *PostgresClient, event stripeapi.Event) error {
	switch event.Type {
	case "checkout.session.completed":
		return handleCheckoutCompleted(ctx, pg, event)
	case "customer.subscription.updated":
		return handleSubscriptionUpdated(ctx, pg, event)
	case "customer.subscription.deleted":
		return handleSubscriptionDeleted(ctx, pg, event)
	case "invoice.payment_failed":
		return handlePaymentFailed(ctx, pg, event)
	}
	return nil
}

func handleCheckoutCompleted(ctx context.Context, pg *PostgresClient, event stripeapi.Event) error {
	var session stripeapi.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		return fmt.Errorf("unmarshal checkout session: %w", err)
	}
	if session.Customer == nil {
		return fmt.Errorf("no customer in checkout session")
	}

	userIDStr := session.Metadata["user_id"]
	var userID int64
	fmt.Sscanf(userIDStr, "%d", &userID)

	user, err := pg.GetUserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	if err := pg.UpdateUserStripeCustomerID(ctx, user.ID, session.Customer.ID); err != nil {
		return fmt.Errorf("update stripe customer id: %w", err)
	}
	if err := pg.UpdateUserPlan(ctx, user.ID, "pro"); err != nil {
		return fmt.Errorf("update user plan: %w", err)
	}
	return nil
}

func handleSubscriptionUpdated(ctx context.Context, pg *PostgresClient, event stripeapi.Event) error {
	var sub stripeapi.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}
	_ = sub
	return nil
}

func handleSubscriptionDeleted(ctx context.Context, pg *PostgresClient, event stripeapi.Event) error {
	var sub stripeapi.Subscription
	if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
		return fmt.Errorf("unmarshal subscription: %w", err)
	}
	if sub.Customer != nil {
		// TODO: find user by stripe customer id and downgrade
		_ = sub.Customer.ID
	}
	return nil
}

func handlePaymentFailed(ctx context.Context, pg *PostgresClient, event stripeapi.Event) error {
	var inv stripeapi.Invoice
	if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
		return fmt.Errorf("unmarshal invoice: %w", err)
	}
	_ = inv
	return nil
}
