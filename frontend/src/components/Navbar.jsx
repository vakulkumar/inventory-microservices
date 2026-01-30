import React from 'react';

export default function Navbar({ cartCount, onCartClick }) {
    return (
        <nav className="navbar">
            <div className="container">
                <div className="navbar-content">
                    <div className="logo">🛍️ ShopHub</div>

                    <div className="search-bar">
                        <input
                            type="text"
                            placeholder="Search for products, brands and more..."
                        />
                    </div>

                    <div className="nav-actions">
                        <button className="cart-button" onClick={onCartClick}>
                            <span>🛒</span>
                            <span>Cart</span>
                            {cartCount > 0 && (
                                <span className="cart-count">{cartCount}</span>
                            )}
                        </button>
                    </div>
                </div>
            </div>
        </nav>
    );
}
