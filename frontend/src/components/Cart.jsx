import React from 'react';

export default function Cart({ cart, onClose, onUpdateQuantity, onRemoveItem, onCheckout, isCheckingOut }) {
    const total = cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);

    if (!cart.length) {
        return (
            <div className="cart-modal-overlay" onClick={onClose}>
                <div className="cart-modal" onClick={e => e.stopPropagation()}>
                    <div className="cart-header">
                        <h2 className="cart-title">Shopping Cart</h2>
                        <button className="close-cart-btn" onClick={onClose}>×</button>
                    </div>
                    <div className="cart-items">
                        <div className="empty-state">
                            <div className="empty-state-icon">🛒</div>
                            <p>Your cart is empty</p>
                            <p style={{ fontSize: '0.875rem', marginTop: '0.5rem' }}>
                                Add some amazing products to get started!
                            </p>
                        </div>
                    </div>
                </div>
            </div>
        );
    }

    return (
        <div className="cart-modal-overlay" onClick={onClose}>
            <div className="cart-modal" onClick={e => e.stopPropagation()}>
                <div className="cart-header">
                    <h2 className="cart-title">Shopping Cart ({cart.length})</h2>
                    <button className="close-cart-btn" onClick={onClose}>×</button>
                </div>

                <div className="cart-items">
                    {cart.map(item => (
                        <div key={item.id} className="cart-item">
                            <div className="cart-item-info">
                                <div className="cart-item-name">{item.name}</div>
                                <div className="cart-item-price">${item.price.toFixed(2)} × {item.quantity}</div>

                                <div className="cart-item-controls">
                                    <button
                                        className="quantity-btn"
                                        onClick={() => onUpdateQuantity(item.id, item.quantity - 1)}
                                        disabled={item.quantity <= 1}
                                    >
                                        −
                                    </button>
                                    <span style={{ fontWeight: 600 }}>{item.quantity}</span>
                                    <button
                                        className="quantity-btn"
                                        onClick={() => onUpdateQuantity(item.id, item.quantity + 1)}
                                    >
                                        +
                                    </button>
                                    <button
                                        className="remove-item-btn"
                                        onClick={() => onRemoveItem(item.id)}
                                    >
                                        Remove
                                    </button>
                                </div>
                            </div>

                            <div style={{
                                fontSize: '1.25rem',
                                fontWeight: 700,
                                color: 'var(--primary)'
                            }}>
                                ${(item.price * item.quantity).toFixed(2)}
                            </div>
                        </div>
                    ))}
                </div>

                <div className="cart-footer">
                    <div className="cart-total">
                        <span>Total:</span>
                        <span>${total.toFixed(2)}</span>
                    </div>
                    <button
                        className="checkout-btn"
                        onClick={onCheckout}
                        disabled={isCheckingOut}
                    >
                        {isCheckingOut ? 'Processing...' : 'Proceed to Checkout'}
                    </button>
                </div>
            </div>
        </div>
    );
}
