package action

import (
	"io"

	"github.com/deislabs/cnab-go/claim"
	"github.com/deislabs/cnab-go/credentials"
	"github.com/deislabs/cnab-go/driver"
)

// Upgrade runs an upgrade action
type Upgrade struct {
	Driver driver.Driver

	// OperationConfig is an optional handler that applies additional
	// configuration to the operation before it is executed.
	OperationConfig func(operation *driver.Operation)
}

// Run performs the upgrade steps and updates the Claim
func (u *Upgrade) Run(c *claim.Claim, creds credentials.Set, w io.Writer) error {
	invocImage, err := selectInvocationImage(u.Driver, c)
	if err != nil {
		return err
	}

	op, err := opFromClaim(claim.ActionUpgrade, stateful, c, invocImage, creds, w)
	if err != nil {
		return err
	}

	if u.OperationConfig != nil {
		u.OperationConfig(op)
	}

	opResult, err := u.Driver.Run(op)
	outputErrors := setOutputsOnClaim(c, opResult.Outputs)

	if err != nil {
		c.Update(claim.ActionUpgrade, claim.StatusFailure)
		c.Result.Message = err.Error()
		return err
	}
	c.Update(claim.ActionUpgrade, claim.StatusSuccess)

	return outputErrors
}
