import React from 'react';

const GRADIENT_COLORS = [
    'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
    'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
    'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
    'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)',
    'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
    'linear-gradient(135deg, #30cfd0 0%, #330867 100%)',
];

export default function ProductCard({ product, onAddToCart }) {
    const isLowStock = product.stock > 0 && product.stock < 10;
    const isOutOfStock = product.stock === 0;

    // Generate consistent color based on product ID
    const gradientColor = GRADIENT_COLORS[product.id % GRADIENT_COLORS.length];

    return (
        <div className="product-card">
            <div
                className="product-image"
                style={{ background: gradientColor }}
            >
                <span>{product.name.charAt(0)}</span>
            </div>

            <div className="product-info">
                <h3 className="product-name">{product.name}</h3>
                <p className="product-description">
                    {product.description || 'Premium quality product for your needs'}
                </p>

                <div className="product-footer">
                    <div>
                        <div className="product-price">
                            ${product.price.toFixed(2)}
                        </div>
                        <div className={`product-stock ${isLowStock ? 'low-stock' : ''} ${isOutOfStock ? 'out-of-stock' : ''}`}>
                            {isOutOfStock ? '❌ Out of Stock' :
                                isLowStock ? `⚠️ Only ${product.stock} left` :
                                    `✅ ${product.stock} in stock`}
                        </div>
                    </div>
                </div>

                <button
                    className="add-to-cart-btn"
                    onClick={() => onAddToCart(product)}
                    disabled={isOutOfStock}
                >
                    {isOutOfStock ? 'Out of Stock' : 'Add to Cart'}
                </button>
            </div>
        </div>
    );
}
